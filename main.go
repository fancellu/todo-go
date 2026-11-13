package main

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Templates and static assets are compiled into the binary so it runs
// from any working directory (including serverless platforms like Vercel).
var (
	//go:embed templates/*.html
	templatesFS embed.FS

	//go:embed static
	staticFS embed.FS
)

const (
	cookieName = "todos"
	// Browsers cap a cookie at about 4 KB, including its name and attributes.
	maxCookieValue = 3800
	cookieMaxAge   = 365 * 24 * 60 * 60 // one year, in seconds
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// taskList is one visitor's todo list. It lives in a cookie in their browser,
// not on the server, so every request carries its own list.
type taskList struct {
	NextID int    `json:"next"`
	Tasks  []Task `json:"tasks"`
}

func main() {
	addrPtr := flag.String("addr", ":8080", "HTTP network address")
	flag.Parse()

	addr := *addrPtr

	if envPort := os.Getenv("PORT"); envPort != "" {
		addr = "0.0.0.0:" + envPort
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Starting server on", addr)
	router := gin.New()
	router.Use(gin.Recovery())

	router.SetHTMLTemplate(template.Must(template.ParseFS(templatesFS, "templates/*.html")))
	router.StaticFS("/static", http.FS(staticSub))
	// Index
	router.GET("/", indexHandler)
	router.POST("/task", createTaskHandler)
	router.GET("/task/:id", getTaskHandler)
	router.PUT("/task/:id", updateTaskHandler)
	router.DELETE("/task/:id", deleteTaskHandler)
	router.PUT("/task/:id/title", updateTaskTitleHandler)

	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
	log.Println("Shutdown server on", addr)
}

// loadTasks reads the visitor's list from the cookie. A missing or unreadable
// cookie gives an empty list.
func loadTasks(c *gin.Context) taskList {
	var tl taskList
	raw, err := c.Cookie(cookieName)
	if err != nil {
		return tl
	}
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || json.Unmarshal(data, &tl) != nil {
		return taskList{}
	}
	return tl
}

// saveTasks writes the list back to the cookie as base64-encoded JSON.
// If the list no longer fits in a cookie it writes a 400 error page and returns false.
func saveTasks(c *gin.Context, tl taskList) bool {
	data, err := json.Marshal(tl)
	if err != nil {
		renderError(c, http.StatusInternalServerError, "Could not save tasks")
		return false
	}
	value := base64.RawURLEncoding.EncodeToString(data)
	if len(value) > maxCookieValue {
		renderError(c, http.StatusBadRequest, "Your list is full. Delete some tasks or shorten their titles.")
		return false
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https",
		SameSite: http.SameSiteLaxMode,
	})
	return true
}

// parseID reads the :id path parameter, writing a 400 error page if it is not an integer.
func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, http.StatusBadRequest, "Invalid task ID: "+c.Param("id"))
		return 0, false
	}
	return id, true
}

// findTask returns the index of the task with the given ID, or -1.
func findTask(tasks []Task, id int) int {
	return slices.IndexFunc(tasks, func(t Task) bool { return t.ID == id })
}

func renderError(c *gin.Context, status int, message string) {
	c.HTML(status, "error.html", gin.H{"message": message})
}

func indexHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"tasks": loadTasks(c).Tasks,
	})
}

func createTaskHandler(c *gin.Context) {
	title := c.PostForm("title")

	if title == "" {
		renderError(c, http.StatusBadRequest, "Title is required")
		return
	}

	tl := loadTasks(c)
	// The cookie is client-editable, so make sure the new ID can't collide.
	for _, t := range tl.Tasks {
		tl.NextID = max(tl.NextID, t.ID+1)
	}
	tl.Tasks = append(tl.Tasks, Task{ID: tl.NextID, Title: title})
	tl.NextID++

	if saveTasks(c, tl) {
		c.Redirect(http.StatusSeeOther, "/")
	}
}

func getTaskHandler(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	tl := loadTasks(c)
	i := findTask(tl.Tasks, id)
	if i < 0 {
		renderError(c, http.StatusNotFound, "Task not found")
		return
	}
	c.HTML(http.StatusOK, "task.html", gin.H{
		"task": tl.Tasks[i],
	})
}

func updateTaskTitleHandler(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	newTitle := c.PostForm("title")

	tl := loadTasks(c)
	i := findTask(tl.Tasks, id)
	if i < 0 {
		renderError(c, http.StatusNotFound, "Task not found")
		return
	}
	tl.Tasks[i].Title = newTitle
	if saveTasks(c, tl) {
		c.Redirect(http.StatusSeeOther, "/")
	}
}

func updateTaskHandler(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	completed := c.PostForm("completed") == "true"

	tl := loadTasks(c)
	i := findTask(tl.Tasks, id)
	if i < 0 {
		renderError(c, http.StatusNotFound, "Task not found")
		return
	}
	tl.Tasks[i].Completed = completed
	if saveTasks(c, tl) {
		c.Redirect(http.StatusSeeOther, "/")
	}
}

func deleteTaskHandler(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	tl := loadTasks(c)
	i := findTask(tl.Tasks, id)
	if i < 0 {
		renderError(c, http.StatusNotFound, "Task not found")
		return
	}
	tl.Tasks = slices.Delete(tl.Tasks, i, i+1)
	if saveTasks(c, tl) {
		c.Redirect(http.StatusSeeOther, "/")
	}
}
