package db

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
	"github.com/stenstromen/miniomatic/model"
)

var (
	// Database instance
	db *sql.DB
	// Database file path
	dbPath = "assets/db.sqlite"
)

// InitDB initializes the SQLite database and the necessary tables
func InitDB() error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	// Create table if it doesn't exist
	query := `
	CREATE TABLE IF NOT EXISTS records (
		status TEXT NOT NULL DEFAULT 'provisioning',
		date TEXT NOT NULL,
		id TEXT PRIMARY KEY,
		init_bucket TEXT NOT NULL,
		url TEXT NOT NULL,
		storage INTEGER NOT NULL
	);
	`

	if _, err := db.Exec(query); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	return nil
}

func UpdateStatus(id, status string) error {
	_, err := db.Exec("UPDATE records SET status = ? WHERE id = ?", status, id)
	if err != nil {
		log.Fatalf("failed to update status: %v", err)
	}
	return nil
}

// InsertData inserts a new record into the database
func InsertData(id, initBucket, storage string) error {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	url := "https://" + id + "." + os.Getenv("WILDCARD_DOMAIN")

	_, err := db.Exec("INSERT INTO records (date, id, init_bucket, url, storage) VALUES (?, ?, ?, ?, ?)", currentTime, id, initBucket, url, storage)
	if err != nil {
		log.Fatalf("failed to insert data: %v", err)
	}
	return nil
}

func UpdateData(id, initBucket, storage string) error {
	_, err := db.Exec("UPDATE records SET init_bucket = ?, storage = ? WHERE id = ?", initBucket, storage, id)
	if err != nil {
		log.Fatalf("failed to update data: %v", err)
	}
	return nil
}

// DeleteData deletes a record by its ID
func DeleteData(id string) error {
	result, err := db.Exec("DELETE FROM records WHERE id = ?", id)
	if err != nil {
		log.Fatalf("failed to delete data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("failed to retrieve rows affected count: %v", err)
	}

	if rowsAffected == 0 {
		log.Fatalf("no record found with ID %s", id)
	}

	return nil
}

func GetAllData() ([]model.Record, error) {
	rows, err := db.Query("SELECT status, date, id, init_bucket, url, storage FROM records")
	if err != nil {
		log.Fatalf("failed to get all data: %v", err)
	}
	defer rows.Close()

	var records []model.Record
	for rows.Next() {
		var r model.Record
		if err := rows.Scan(&r.Status, &r.Date, &r.ID, &r.InitBucket, &r.URL, &r.Storage); err != nil {
			return nil, err
		}
		records = append(records, r)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// GetDataByID retrieves a specific record by its ID
func GetDataByID(id string) (*model.Record, error) {
	row := db.QueryRow("SELECT status, date, id, init_bucket, url, storage FROM records WHERE id = ?", id)

	var r model.Record
	if err := row.Scan(&r.Status, &r.Date, &r.ID, &r.InitBucket, &r.URL, &r.Storage); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No data found for the given ID
		}
		log.Fatalf("failed to get data by ID: %v", err)
	}

	return &r, nil
}
