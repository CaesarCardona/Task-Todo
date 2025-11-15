package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/segmentio/kafka-go"
	"github.com/rs/cors"
	_ "github.com/mattn/go-sqlite3"
)

type Task struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	db          *sql.DB
	kafkaWriter *kafka.Writer
)

func main() {
	var err error

	// --- SQLite setup ---
	db, err = sql.Open("sqlite3", "./tasks.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}

	// --- Kafka setup ---
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "tasks",
		Balancer: &kafka.LeastBytes{},
	}

	// --- Start Kafka consumer in background ---
	go consumeKafkaMessages()

	// --- HTTP routes ---
	http.HandleFunc("/tasks", getTasks)
	http.HandleFunc("/add", addTask)
	http.HandleFunc("/delete", deleteTask)

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "DELETE"},
	}).Handler(http.DefaultServeMux)

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}

// --- Kafka consumer ---
func consumeKafkaMessages() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       "tasks",
		GroupID:     "console-consumer",
		StartOffset: kafka.FirstOffset, // read from beginning
	})

	fmt.Println("Kafka consumer running...")

	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Println("Error reading Kafka message:", err)
			continue
		}
		fmt.Printf("[Kafka Message] Key: %s | Value: %s\n", string(msg.Key), string(msg.Value))
	}
}

// --- HTTP Handlers ---

func getTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name FROM tasks")
	if err != nil {
		http.Error(w, "Error fetching tasks", 500)
		return
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			http.Error(w, "Error scanning task", 500)
			return
		}
		tasks = append(tasks, t)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func addTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	res, err := db.Exec("INSERT INTO tasks(name) VALUES(?)", t.Name)
	if err != nil {
		http.Error(w, "Error inserting task", 500)
		return
	}

	id, _ := res.LastInsertId()
	t.ID = int(id)

	// Produce to Kafka
	data, _ := json.Marshal(t)
	_ = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", t.ID)),
		Value: data,
	})

	fmt.Fprint(w, "Task added successfully")
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id", 400)
		return
	}

	res, err := db.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		http.Error(w, "Error deleting task", 500)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Task not found", 404)
		return
	}

	// Produce deletion to Kafka
	taskDel := Task{ID: atoi(id), Name: ""}
	data, _ := json.Marshal(taskDel)
	_ = kafkaWriter.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte(id),
		Value: data,
	})

	fmt.Fprint(w, "Task deleted successfully")
}

// --- Helper ---
func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

