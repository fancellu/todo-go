package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
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

	log.Println("Starting server on", addr)
	router := gin.New()
	router.Use(gin.Recovery())

	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")
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
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"message": "Title is required",
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()
	var newTask Task = Task{
		Title: title,
	}
	newTask.ID = nextID
	nextID++
	tasks = append(tasks, newTask)

	c.Redirect(http.StatusSeeOther, "/")
}

func getTaskHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	_, err := fmt.Sscan(idStr, &id)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"message": fmt.Sprintf("Invalid task ID: %v", err),
		})
		return
	}

	mu.RLock()
	defer mu.RUnlock()

	for _, task := range tasks {
		if task.ID == id {
			c.HTML(http.StatusOK, "task.html", gin.H{
				"task": task,
			})
			return
		}
	}

	c.HTML(http.StatusNotFound, "error.html", gin.H{
		"message": "Task not found",
	})
}

func updateTaskTitleHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	_, err := fmt.Sscan(idStr, &id)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"message": fmt.Sprintf("Invalid task ID: %v", err),
		})
		return
	}

	newTitle := c.PostForm("title")

	mu.Lock()
	defer mu.Unlock()
	for i, task := range tasks {
		if task.ID == id {
			tasks[i].Title = newTitle
			c.Redirect(http.StatusSeeOther, "/")
			return
		}
	}

	c.HTML(http.StatusNotFound, "error.html", gin.H{
		"message": "Task not found",
	})

}

func updateTaskHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	_, err := fmt.Sscan(idStr, &id)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"message": fmt.Sprintf("Invalid task ID: %v", err),
		})
		return
	}

	completedStr := c.PostForm("completed")
	completed := completedStr == "true"

	mu.Lock()
	defer mu.Unlock()
	for i, task := range tasks {
		if task.ID == id {
			tasks[i].Completed = completed
			c.Redirect(http.StatusSeeOther, "/")
			return
		}
	}

	c.HTML(http.StatusNotFound, "error.html", gin.H{
		"message": "Task not found",
	})

}

func deleteTaskHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	_, err := fmt.Sscan(idStr, &id)
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{
			"message": fmt.Sprintf("Invalid task ID: %v", err),
		})
		return
	}

	mu.Lock()
	defer mu.Unlock()

	for i, task := range tasks {
		if task.ID == id {
			tasks = append(tasks[:i], tasks[i+1:]...)
			c.Redirect(http.StatusSeeOther, "/")
			return
		}
	}
	c.HTML(http.StatusNotFound, "error.html", gin.H{
		"message": "Task not found",
	})
}
