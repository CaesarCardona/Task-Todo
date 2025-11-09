## 📋 Task Manager

A simple Go + SQLite3 backend with a lightweight HTML frontend.



## 🚀 Requirements

Go (version ≥ 1.18 recommended)
SQLite3

# Usage

## ⚙️Execution
### Backend setup.
```bash
cd backend
go get github.com/mattn/go-sqlite3
go run main.go
```
### Frontend setup.

Open index.html

## To read database queries.
```bash
sqlite3 tasks.db
```

Show all tables and schema of tasks table.
```bash
.tables
.schema tasks
```

See all tasks

```bash
 SELECT * FROM tasks;
```

Delete task 3 
```bash
DELETE FROM tasks WHERE id = 3;
```
Instert new task
```bash
INSERT INTO tasks(name) VALUES('Purge Cache');
```


