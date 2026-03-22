package repository

import (
	"database/sql"
	"kate/services/tasks/internal/service"
	"time"

	_ "modernc.org/sqlite"
)

type TaskRepository interface {
	Create(task service.Task) (service.Task, error)
	GetAll() ([]service.Task, error)
	GetByID(id string) (service.Task, error)
	Update(id string, task service.Task) (service.Task, error)
	Delete(id string) error
	SearchByTitle(title string) ([]service.Task, error)
	InitDB() error
}

type SQLiteTaskRepository struct {
	db *sql.DB
}

func NewSQLiteTaskRepository(dbPath string) (*SQLiteTaskRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	return &SQLiteTaskRepository{db: db}, nil
}

func (r *SQLiteTaskRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteTaskRepository) InitDB() error {
	query := `
    CREATE TABLE IF NOT EXISTS tasks (
        id TEXT PRIMARY KEY,
        title TEXT NOT NULL,
        description TEXT,
        due_date TEXT,
        done INTEGER DEFAULT 0,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );
    `
	_, err := r.db.Exec(query)
	return err
}

func (r *SQLiteTaskRepository) Create(task service.Task) (service.Task, error) {
	query := `INSERT INTO tasks (id, title, description, due_date, done, created_at) VALUES (?, ?, ?, ?, ?, ?)`
	task.ID = "t_" + time.Now().Format("150405.000")
	task.CreatedAt = time.Now()
	_, err := r.db.Exec(query, task.ID, task.Title, task.Description, task.DueDate, 0, task.CreatedAt)
	return task, err
}

func (r *SQLiteTaskRepository) GetAll() ([]service.Task, error) {
	query := `SELECT id, title, description, due_date, done, created_at FROM tasks`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []service.Task
	for rows.Next() {
		var t service.Task
		var doneInt int
		rows.Scan(&t.ID, &t.Title, &t.Description, &t.DueDate, &doneInt, &t.CreatedAt)
		t.Done = doneInt == 1
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *SQLiteTaskRepository) GetByID(id string) (service.Task, error) {
	query := `SELECT id, title, description, due_date, done, created_at FROM tasks WHERE id = ?`
	var t service.Task
	var doneInt int
	err := r.db.QueryRow(query, id).Scan(&t.ID, &t.Title, &t.Description, &t.DueDate, &doneInt, &t.CreatedAt)
	t.Done = doneInt == 1
	return t, err
}

func (r *SQLiteTaskRepository) Update(id string, updated service.Task) (service.Task, error) {
	t, err := r.GetByID(id)
	if err != nil {
		return service.Task{}, err
	}
	if updated.Title != "" {
		t.Title = updated.Title
	}
	if updated.Description != "" {
		t.Description = updated.Description
	}
	if updated.DueDate != "" {
		t.DueDate = updated.DueDate
	}
	t.Done = updated.Done

	doneInt := 0
	if t.Done {
		doneInt = 1
	}
	query := `UPDATE tasks SET title = ?, description = ?, due_date = ?, done = ? WHERE id = ?`
	_, err = r.db.Exec(query, t.Title, t.Description, t.DueDate, doneInt, id)
	return t, err
}

func (r *SQLiteTaskRepository) Delete(id string) error {
	query := `DELETE FROM tasks WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *SQLiteTaskRepository) SearchByTitle(title string) ([]service.Task, error) {

	println("SEARCH CALLED with title:", title)

	query := `SELECT id, title, description, due_date, done, created_at FROM tasks WHERE title LIKE ?`
	rows, err := r.db.Query(query, "%"+title+"%")
	if err != nil {
		println("QUERY ERROR:", err.Error())
		return nil, err
	}
	defer rows.Close()

	var tasks []service.Task
	for rows.Next() {
		var t service.Task
		var doneInt int
		err := rows.Scan(&t.ID, &t.Title, &t.Description, &t.DueDate, &doneInt, &t.CreatedAt)
		if err != nil {
			println("SCAN ERROR:", err.Error())
			return nil, err
		}
		t.Done = doneInt == 1
		tasks = append(tasks, t)
		println("FOUND:", t.Title)
	}

	println("TOTAL FOUND:", len(tasks))
	return tasks, nil
}
