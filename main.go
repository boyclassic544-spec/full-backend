package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

var db *sql.DB

// 1. Muundo wa Tukio (Event Struct)
type Event struct {
	ID        int    `json:"id"`
	EventName string `json:"event_name"`
}

// 2. Muundo wa Ahadi / Mchango (Pledge Struct)
type Pledge struct {
	ID            int     `json:"id"`
	EventID       int     `json:"event_id"`
	DonorName     string  `json:"donor_name"`
	PhoneNumber   string  `json:"phone_number"`
	PledgedAmount float64 `json:"pledged_amount"`
	Status        string  `json:"status"`
}

func main() {
	// Kuchukua Connection URL ya Database kutoka kwenye Railway Environment Variables
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Kama ipo kwenye jaribio la kienyeji (local), unaweza kuweka connection string yako hapa
		connStr = "postgres://postgres:password@localhost:5432/railway?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Imeshindwa kuunganisha database: ", err)
	}
	defer db.Close()

	// Hakikisha muunganisho upo sawa
	err = db.Ping()
	if err != nil {
		log.Fatal("Database haijibu: ", err)
	}
	fmt.Println("Database imeunganishwa kikamilifu!")

	// Routes za API zetu
	http.HandleFunc("/events", handleEvents)
	http.HandleFunc("/pledges", handlePledges)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Println("Server inaanza kusikiliza kwenye port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Handler ya Kutengeneza Tukio (POST /events)
func handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var event Event
		err := json.NewDecoder(r.Body).Decode(&event)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = db.QueryRow("INSERT INTO events (event_name) VALUES ($1) RETURNING id", event.EventName).Scan(&event.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	} else {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
	}
}

// Handler ya Kuongeza Ahadi ya Mchango (POST /pledges)
func handlePledges(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var p Pledge
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		query := `INSERT INTO pledges (event_id, donor_name, phone_number, pledged_amount, status) VALUES ($1, $2, $3, $4, 'Pending') RETURNING id`
		err = db.QueryRow(query, p.EventID, p.DonorName, p.PhoneNumber, p.PledgedAmount).Scan(&p.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	} else {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
	}
}

