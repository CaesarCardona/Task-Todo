package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/cors"
)

var db *sql.DB

// Task represents a task to be stored in the database
type Task struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func main() {
	var err error
	// Open or create the database
	db, err = sql.Open("sqlite3", "./tasks.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create the tasks table if it doesn't exist
	createTableQuery := `CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal(err)
	}

	// Set up the routes
	http.HandleFunc("/tasks", getTasks)
	http.HandleFunc("/add", addTask)

	// Enable CORS
	handler := cors.Default().Handler(http.DefaultServeMux)

	// Start the server with CORS enabled
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// getTasks handles GET requests to fetch all tasks
func getTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	rows, err := db.Query("SELECT id, name FROM tasks")
	if err != nil {
		http.Error(w, "Error fetching tasks from database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.ID, &task.Name); err != nil {
			http.Error(w, "Error scanning task", http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, task)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// addTask handles POST requests to add a new task
func addTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, "Error decoding task", http.StatusBadRequest)
		return
	}

	// Insert the new task into the database
	stmt, err := db.Prepare("INSERT INTO tasks(name) VALUES(?)")
	if err != nil {
		http.Error(w, "Error preparing query", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(task.Name)
	if err != nil {
		http.Error(w, "Error inserting task into database", http.StatusInternalServerError)
		return
	}

	// Send success response
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Task added successfully")
}

