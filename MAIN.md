# How main.go Works

This document walks through [main.go](main.go): how it starts a web server, answers requests, and connects to the HTML in [templates/](templates/) and the JavaScript in [static/js/](static/js/). It assumes you can read a little Go but have never used Gin.

## What is Gin?

[Gin](https://gin-gonic.com/) is a web framework for Go. Go's standard library can already run a web server (the `net/http` package), and Gin is a layer on top of it that makes the common jobs shorter:

- **Routing:** "when a `PUT` request arrives for `/task/3`, call this function", with `3` pulled out of the URL for you.
- **A context object:** every handler function receives a `*gin.Context` (written `c` by convention). It holds the incoming request and has helpers for reading form fields and sending responses.
- **Middleware:** functions that run around every request, such as logging or crash recovery.
- **HTML rendering:** fill in a template with data and send it as the response.

Everything in this app could be written with plain `net/http`. Gin just removes the boilerplate.

## The big picture

The server renders full HTML pages. There is no separate frontend app and no JSON API. The browser either submits a normal form or makes a small `fetch` call, then reloads the page to see the result.

```mermaid
sequenceDiagram
    participant B as Browser
    participant G as Gin router
    participant H as Handler function
    participant S as tasks slice (memory)

    B->>G: GET /
    G->>H: indexHandler
    H->>S: read tasks
    H-->>B: index.html filled in with tasks

    B->>G: PUT /task/3 (fetch from tasks.js)
    G->>H: updateTaskHandler
    H->>S: set Completed = true
    H-->>B: 303 redirect to /
    Note over B: tasks.js calls location.reload(),<br/>which does GET / again
```

## Startup: `main()`

### 1. Choosing the address

```go
addrPtr := flag.String("addr", ":8080", "HTTP network address")
flag.Parse()

addr := *addrPtr

if envPort := os.Getenv("PORT"); envPort != "" {
    addr = "0.0.0.0:" + envPort
}
```

By default the server listens on port 8080. You can override that with `-addr :9000` on the command line. If the `PORT` environment variable is set, it wins. Hosting platforms such as Vercel tell your app which port to use through `PORT`.

### 2. Creating the router and middleware

```go
router := gin.New()
router.Use(gin.Recovery())
```

`gin.New()` creates an empty router. `router.Use(...)` adds middleware that runs on every request. `gin.Recovery()` catches a panic (Go's equivalent of an uncaught exception) inside a handler and turns it into a `500 Internal Server Error`, instead of letting it crash the whole server.

You may see `gin.Default()` in Gin tutorials. It is `gin.New()` plus `Recovery` plus a request logger. This app leaves the logger out, so it does not print a line per request. When it starts, Gin prints `[GIN-debug]` lines listing every route. Set `GIN_MODE=release` to silence them.

### 3. Embedding the templates and static files

```go
var (
    //go:embed templates/*.html
    templatesFS embed.FS

    //go:embed static
    staticFS embed.FS
)
```

The `//go:embed` comments are instructions to the Go compiler: "copy these files into the program". At runtime, `templatesFS` and `staticFS` behave like read-only folders, but they live inside the binary. That is why the compiled program runs from any directory, or on a server where the `templates/` folder doesn't exist on disk.

They are then handed to Gin:

```go
staticSub, err := fs.Sub(staticFS, "static")
...
router.SetHTMLTemplate(template.Must(template.ParseFS(templatesFS, "templates/*.html")))
router.StaticFS("/static", http.FS(staticSub))
```

- `template.ParseFS` reads every `templates/*.html` file and parses it as a Go template. Each one is registered under its file name (`index.html`, `task.html`, `error.html`). `template.Must` stops the program at startup if a template has a syntax error, so a broken template never reaches users.
- `SetHTMLTemplate` tells Gin to use those templates whenever a handler calls `c.HTML(...)`.
- `fs.Sub(staticFS, "static")` drops the leading `static/` folder name. `StaticFS("/static", ...)` then serves those files as-is, so a request for `/static/js/main.js` returns the embedded file `static/js/main.js`.

### 4. The routes

```go
router.GET("/", indexHandler)
router.POST("/task", createTaskHandler)
router.GET("/task/:id", getTaskHandler)
router.PUT("/task/:id", updateTaskHandler)
router.DELETE("/task/:id", deleteTaskHandler)
router.PUT("/task/:id/title", updateTaskTitleHandler)
```

Each line connects an HTTP method and a URL pattern to a function. `:id` is a **path parameter**: it matches any single segment (`/task/3`, `/task/abc`), and the handler reads it with `c.Param("id")`.

| Method | Path | Handler | Called by | Response |
| --- | --- | --- | --- | --- |
| GET | `/` | `indexHandler` | Browser opening the page | `index.html` with all tasks |
| POST | `/task` | `createTaskHandler` | The "Add Task" form | 303 redirect to `/` |
| GET | `/task/:id` | `getTaskHandler` | Visiting e.g. `/task/3` directly | `task.html` for one task |
| PUT | `/task/:id` | `updateTaskHandler` | Check / undo button (`tasks.js`) | 303 redirect to `/` |
| PUT | `/task/:id/title` | `updateTaskTitleHandler` | Edit save button or Enter (`tasks.js`) | 303 redirect to `/` |
| DELETE | `/task/:id` | `deleteTaskHandler` | Trash button (`tasks.js`) | 303 redirect to `/` |
| GET | `/static/*` | built in (`StaticFS`) | `<script>` tags | The JS file |

When something goes wrong, the handlers send `error.html` instead: 400 for a bad ID or empty title, 404 for an unknown task.

### 5. Starting the server

```go
if err := router.Run(addr); err != nil {
    log.Fatal(err)
}
```

`router.Run` starts listening and blocks forever, serving requests. It only returns if something fails, for example if the port is already in use.

## In-memory state

```go
var (
    mu     sync.RWMutex
    tasks  []Task
    nextID int
)

type Task struct {
    ID        int    `json:"id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
}
```

There is no database. All tasks live in the `tasks` slice (Go's growable array), and `nextID` hands out IDs. Stopping the server loses everything.

Gin handles each request on its own goroutine, Go's lightweight thread, so two requests can run at the same moment. `mu` is a read/write lock that stops them from corrupting the slice:

- `mu.RLock()` is for handlers that only **read** (`indexHandler`, `getTaskHandler`). Many readers can hold it at once.
- `mu.Lock()` is for handlers that **change** tasks. It waits until nobody else holds the lock.
- `defer mu.Unlock()` runs the unlock automatically when the function returns, whichever `return` it leaves through.

The `json:"..."` tags on `Task` are unused today. They would control the field names if a task were ever sent as JSON.

## The handlers

Every handler has the same signature, `func(c *gin.Context)`. These are the `Context` methods the app uses:

| Method | What it does |
| --- | --- |
| `c.Param("id")` | Reads the `:id` part of the URL, as a string |
| `c.PostForm("title")` | Reads a field from a form-encoded request body |
| `c.HTML(status, "name.html", data)` | Renders a template with `data` and sends it |
| `c.Redirect(http.StatusSeeOther, "/")` | Sends a 303 telling the browser to go to `/` |

Three small helpers keep the handlers short:

- **`parseID(c)`** converts `:id` to an `int` with `strconv.Atoi`. If that fails (`/task/abc`), it sends a 400 error page and returns `ok == false`, and the handler just returns.
- **`findTask(id)`** returns the slice index of the task with that ID, or `-1`. The caller must already hold the lock.
- **`renderError(c, status, message)`** renders `error.html` with a message.

The handlers then follow one pattern: parse the input, take the lock, find the task, change it, redirect. For example:

```go
func updateTaskHandler(c *gin.Context) {
    id, ok := parseID(c)
    if !ok {
        return
    }

    completed := c.PostForm("completed") == "true"

    mu.Lock()
    defer mu.Unlock()

    i := findTask(id)
    if i < 0 {
        renderError(c, http.StatusNotFound, "Task not found")
        return
    }
    tasks[i].Completed = completed
    c.Redirect(http.StatusSeeOther, "/")
}
```

**Why redirect instead of returning a page?** This is the *Post/Redirect/Get* pattern. After a change, the server says "now go and GET `/`" (303 See Other). The browser ends up on a normal page load, so pressing refresh never resubmits the change.

## Templates: from `c.HTML` to the page

`indexHandler` is the whole bridge between Go and HTML:

```go
c.HTML(http.StatusOK, "index.html", gin.H{
    "tasks": tasks,
})
```

`gin.H` is just a shorthand for `map[string]any`, a dictionary of values. Gin looks up the template named `index.html` (registered at startup) and runs it with that map as its data. Inside the template, `.` means "the data". So `.tasks` is the slice of tasks.

The relevant part of [templates/index.html](templates/index.html):

```html
{{range .tasks}}
<li class="list-group-item ...">
    {{if .Completed}}
    <i class="bi bi-check-circle-fill text-success"></i>
    {{else}}
    <i class="bi bi-circle text-secondary"></i>
    {{end}}
    <span class="ms-2" {{if .Completed}} style="text-decoration: line-through;" {{end}}>{{.Title}}</span>
    ...
    <button class="... toggle-complete-button"
            data-task-id="{{.ID}}" data-completed="{{.Completed}}">
    ...
</li>
{{end}}
```

- `{{range .tasks}} ... {{end}}` repeats the block once per task. Inside the loop, `.` is now the current `Task`, so `.Title`, `.ID` and `.Completed` are its fields.
- `{{if .Completed}} ... {{else}} ... {{end}}` chooses the icon and adds the strikethrough.
- `{{.Title}}` prints the title. Go's `html/template` **escapes it automatically**, so a task titled `<script>` shows up as text and can't run as code.
- `data-task-id="{{.ID}}"` and `data-completed="{{.Completed}}"` don't change how the page looks. They exist so the JavaScript knows which task a button belongs to. This is the handoff from server to browser.

The other two templates:

- [templates/task.html](templates/task.html) is rendered by `getTaskHandler` with `gin.H{"task": tasks[i]}`, so it uses `.task.Title`, `.task.Completed`, and so on.
- [templates/error.html](templates/error.html) is rendered by `renderError` and shows `{{.message}}`.

Styling comes from Bootstrap and Bootstrap Icons, loaded from a CDN in each template's `<head>`. The Go code doesn't involve them at all.

## The JavaScript in `static/js`

`index.html` loads the scripts at the bottom of the page, so the HTML elements already exist when they run:

```html
<script src="/static/js/main.js"></script>
<script src="/static/js/theme.js"></script>
<script src="/static/js/tasks.js"></script>
```

Those `/static/...` URLs are served by the `StaticFS` route from startup.

### [main.js](static/js/main.js): form helpers

- Stops the "Add Task" form from submitting a title that is only spaces. The input's `required` attribute already blocks a completely empty one. The server checks for empty titles too.
- On the task detail page, submits the checkbox's form when it is ticked.

Adding a task is a plain HTML form, `<form action="/task" method="POST">`, so it needs no JavaScript to reach the server. The browser posts the `title` field, `createTaskHandler` appends the task, and the 303 brings the browser back to `/`.

### [tasks.js](static/js/tasks.js): edit, complete and delete

HTML forms can only send `GET` and `POST`, but the routes for changing a task use `PUT` and `DELETE`. So these buttons use `fetch` instead:

```js
function toggleTaskCompletion(taskId, completed) {
    fetch(`/task/${taskId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: `completed=${completed}`
    })
    .then(response => {
        if (response.ok) {
            window.location.reload();
        }
    })
    ...
}
```

- The task ID comes from the button's `data-task-id` attribute, which the template filled in.
- The body is sent in the same `name=value` format a form uses, so the handler can read it with `c.PostForm("completed")`.
- `fetch` follows the server's 303 redirect by itself (it fetches `/` behind the scenes), sees a 200, and the script reloads the page so the new state appears.

The pencil button doesn't call the server. It hides the title `<span>` and shows the hidden `<input>` and save button that the template already put in the row. Pressing Enter or clicking save calls `updateTaskTitle`, which sends `PUT /task/:id/title`. The trash button sends `DELETE /task/:id`.

### [theme.js](static/js/theme.js): light/dark theme

This one never talks to the server. It adds or removes a `dark` class on the `<html>` element, which the `html.dark` CSS rules in `index.html` style. It remembers the choice in `localStorage` so the theme survives a reload.

The saved theme is first applied by a tiny inline `<script>` in `index.html`'s `<head>`, not by `theme.js`. Scripts at the bottom of the page only run after the browser has started drawing. On a slow connection the page would flash white before switching to dark. The `<head>` script runs before the first paint, and `theme.js` then only syncs the sun/moon icon and handles clicks.

## One click, start to finish: completing a task

1. The browser loads `/`. `indexHandler` renders the list, and the task with ID 3 gets a button with `data-task-id="3" data-completed="false"`.
2. You click the check button. `tasks.js` reads those attributes and calls `toggleTaskCompletion("3", "true")`.
3. That sends `PUT /task/3` with the body `completed=true`.
4. Gin matches the route `PUT /task/:id` and calls `updateTaskHandler`.
5. The handler parses `3`, takes the write lock, sets `tasks[i].Completed = true`, and responds `303 See Other` with `Location: /`.
6. `fetch` follows the redirect and gets a 200, so `tasks.js` calls `window.location.reload()`.
7. The reload is a fresh `GET /`. `indexHandler` renders again, and now `{{if .Completed}}` is true, so the title has a strikethrough.

## Known quirks

- **The task detail page's checkbox doesn't work.** The form in `task.html` posts to `/task/:id` with `method="POST"`, but only `PUT` is routed for that path, so ticking it returns a 404. (The hidden `_method=PUT` field is a convention some frameworks use to fake a `PUT`. Gin doesn't support it out of the box.)
- **Tasks are global.** Everyone using the same server sees and edits the same list. On Vercel, the list resets whenever the instance is recycled.

## Where to go next

- [Gin documentation](https://gin-gonic.com/en/docs/)
- [`html/template`](https://pkg.go.dev/html/template), the template language used in `templates/`
- [`embed`](https://pkg.go.dev/embed), how `//go:embed` works
- [`sync.RWMutex`](https://pkg.go.dev/sync#RWMutex)
