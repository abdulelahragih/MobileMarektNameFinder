package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type RequestBody struct {
	RetailBranding string `json:"retail_branding"`
	Model          string `json:"model"`
}

type ResponseData struct {
	Data string `json:"data"`
}

func main() {
	db, err := sql.Open("sqlite3", "./devices.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize the database
	if err := initializeDatabase(db); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}

	http.HandleFunc("/get-device-name", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		var reqBody RequestBody
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&reqBody); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}

		var marketingName string
		query := `SELECT marketing_name FROM devices WHERE retail_branding = ? AND model = ? LIMIT 1`
		err := db.QueryRow(query, reqBody.RetailBranding, reqBody.Model).Scan(&marketingName)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Device not found", http.StatusNotFound)
			} else {
				http.Error(w, "Database query error", http.StatusInternalServerError)
			}
			return
		}

		response := ResponseData{Data: marketingName}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/update-devices", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
			return
		}

		err := updateDevices(db)
		if err != nil {
			log.Printf("Failed to update devices: %v", err)
			http.Error(w, "Failed to update devices", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Devices updated successfully"))
	})

	log.Println("Server started at :8089")
	if err := http.ListenAndServe(":8089", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func initializeDatabase(db *sql.DB) error {
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS devices (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        retail_branding TEXT NOT NULL,
        marketing_name TEXT NOT NULL,
        model TEXT NOT NULL,
        UNIQUE(retail_branding, model)
    );`
	_, err := db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create devices table: %v", err)
	}
	return nil
}

func updateDevices(db *sql.DB) error {
	csvURL := "https://storage.googleapis.com/play_public/supported_devices.csv"

	// Fetch the CSV data
	resp, err := http.Get(csvURL)
	if err != nil {
		return fmt.Errorf("failed to fetch CSV: %v", err)
	}
	defer resp.Body.Close()

	// Skip the first line (license information)
	reader := bufio.NewReader(resp.Body)
	_, err = reader.ReadString('\n') // skip license line
	if err != nil {
		return fmt.Errorf("failed to read license line: %v", err)
	}

	// Read the CSV data
	csvReader := csv.NewReader(reader)
	// Read the header line
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %v", err)
	}

	// Map header indices
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[h] = i
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO devices (retail_branding, marketing_name, model) VALUES (?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	count := 0
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to read CSV record: %v", err)
		}

		retailBranding := record[headerMap["Retail Branding"]]
		marketingName := record[headerMap["Marketing Name"]]
		model := record[headerMap["Model"]]

		_, err = stmt.Exec(retailBranding, marketingName, model)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute statement: %v", err)
		}
		count++
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Inserted or ignored %d records", count)
	return nil
}
