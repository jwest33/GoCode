package main

import (
	"net/http"
	"task-api/internal/database"
	"task-api/internal/handlers"
	"time"
)

func main() {
	// Initialize database connection
	db, err := database.InitDB()
	if err != nil {
		panic(err)
	}

	defer db.Close()

	// Set up handlers
	http.HandleFunc("/api/tasks", handlers.GetTasks)
	http.HandleFunc("/api/tasks/", handlers.GetTask)
	http.HandleFunc("/api/tasks", handlers.CreateTask)
	http.HandleFunc("/api/tasks/", handlers.UpdateTask)
	http.HandleFunc("/api/tasks/", handlers.DeleteTask)

	// Start server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      http.DefaultServeMux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	println("Starting server on :8080")
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}