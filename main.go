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
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"

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
		err := db.QueryRow(query, strings.ToLower(reqBody.RetailBranding), strings.ToLower(reqBody.Model)).Scan(&marketingName)
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
			http.Error(w, fmt.Sprintf("Failed to update devices: %v", err), http.StatusInternalServerError)
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

	// Decode UTF-16 LE to UTF-8
	unicodeReader := transform.NewReader(resp.Body, unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder())

	// Read the CSV data
	reader := bufio.NewReader(unicodeReader)

	// Peek at the first character to check for a license line
	firstLinePeek, err := reader.Peek(1)
	if err != nil {
		return fmt.Errorf("failed to peek the first byte: %v", err)
	}

	if firstLinePeek[0] == '#' {
		// It's a license line, skip it
		licenseLine, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read license line: %v", err)
		}
		log.Printf("License line: %s", strings.TrimSpace(licenseLine))
	}

	csvReader := csv.NewReader(reader)
	csvReader.LazyQuotes = true
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields per record

	// Read the header line
	header, err := csvReader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %v", err)
	}

	// Print the header line for debugging
	log.Printf("CSV Header: %v", header)

	// Map header indices with trimming and case-insensitive matching
	headerMap := make(map[string]int)
	for i, h := range header {
		trimmedHeader := strings.TrimSpace(h)
		lowerHeader := strings.ToLower(trimmedHeader)
		headerMap[lowerHeader] = i
	}

	// Expected header field names (in lowercase)
	expectedFields := []string{"retail branding", "marketing name", "model"}

	// Check if required headers are present
	indices := make(map[string]int)
	missingFields := []string{}
	for _, fieldName := range expectedFields {
		if index, ok := headerMap[fieldName]; ok {
			indices[fieldName] = index
		} else {
			missingFields = append(missingFields, fieldName)
		}
	}

	if len(missingFields) > 0 {
		log.Printf("Required header fields not found: %v", missingFields)
		log.Printf("Available header fields: %v", headerMap)
		return fmt.Errorf("required header fields not found: %v", missingFields)
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
	recordNumber := 1 // Start at 1 because we've read the header
	for {
		record, err := csvReader.Read()
		recordNumber++
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: Failed to read CSV record on line %d: %v", recordNumber, err)
			continue // Skip this record and continue with the next
		}

		// Ensure required fields are present
		if len(record) <= indices["retail branding"] || len(record) <= indices["marketing name"] || len(record) <= indices["model"] {
			log.Printf("Warning: Record on line %d does not have required fields", recordNumber)
			continue // Skip this record and continue with the next
		}

		retailBranding := strings.ToLower(record[indices["retail branding"]])
		marketingName := record[indices["marketing name"]]
		model := strings.ToLower(record[indices["model"]])

		_, err = stmt.Exec(retailBranding, marketingName, model)
		if err != nil {
			log.Printf("Warning: Failed to insert record on line %d: %v", recordNumber, err)
			continue // Skip this record and continue with the next
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
