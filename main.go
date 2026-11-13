package main

import (
	"embed"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"

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

var (
	// mutex for safe access
	mu     sync.RWMutex
	tasks  []Task
	nextID int
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
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

// parseID reads the :id path parameter, writing a 400 error page if it is not an integer.
func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		renderError(c, http.StatusBadRequest, "Invalid task ID: "+c.Param("id"))
		return 0, false
	}
	return id, true
}

// findTask returns the index of the task with the given ID, or -1. Caller must hold mu.
func findTask(id int) int {
	return slices.IndexFunc(tasks, func(t Task) bool { return t.ID == id })
}

func renderError(c *gin.Context, status int, message string) {
	c.HTML(status, "error.html", gin.H{"message": message})
}

func indexHandler(c *gin.Context) {
	mu.RLock()
	defer mu.RUnlock()
	c.HTML(http.StatusOK, "index.html", gin.H{
		"tasks": tasks,
	})
}

func createTaskHandler(c *gin.Context) {
	title := c.PostForm("title")

	if title == "" {
		renderError(c, http.StatusBadRequest, "Title is required")
		return
	}

	mu.Lock()
	defer mu.Unlock()
	tasks = append(tasks, Task{ID: nextID, Title: title})
	nextID++

	c.Redirect(http.StatusSeeOther, "/")
}

func getTaskHandler(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	i := findTask(id)
	if i < 0 {
		renderError(c, http.StatusNotFound, "Task not found")
		return
	}
	c.HTML(http.StatusOK, "task.html", gin.H{
		"task": tasks[i],
	})
}

func updateTaskTitleHandler(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	newTitle := c.PostForm("title")

	mu.Lock()
	defer mu.Unlock()

	i := findTask(id)
	if i < 0 {
		renderError(c, http.StatusNotFound, "Task not found")
		return
	}
	tasks[i].Title = newTitle
	c.Redirect(http.StatusSeeOther, "/")
}

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

func deleteTaskHandler(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	i := findTask(id)
	if i < 0 {
		renderError(c, http.StatusNotFound, "Task not found")
		return
	}
	tasks = slices.Delete(tasks, i, i+1)
	c.Redirect(http.StatusSeeOther, "/")
}
