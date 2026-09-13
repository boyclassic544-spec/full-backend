package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"text/template"

	_ "github.com/lib/pq"
)

var db *sql.DB

type Design struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
	Category    string  `json:"category"`
}

func main() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL haijapatikana kwenye Railway environment variables!")
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Imeshindikana kuunganisha na Database: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Database haijibu vizuri: %v", err)
	}
	fmt.Println("Umeunganishwa kikamilifu na PostgreSQL kwenye Railway!")

	// Inajitengenezea Tables zenyewe kimyakimya kama hazipo
	initDatabase()

	// Routes za Mfumo
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/designs", designsHandler)
	http.HandleFunc("/api/upload", uploadHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Seva ya full-backend inaendelea kusikiliza kwenye port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDatabase() {
	queryDesigns := `
	CREATE TABLE IF NOT EXISTS designs (
		id SERIAL PRIMARY KEY,
		title VARCHAR(150) NOT NULL,
		description TEXT,
		price DECIMAL(10, 2) NOT NULL,
		image_url TEXT NOT NULL,
		category VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	queryOrders := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		design_id INT REFERENCES designs(id) ON DELETE CASCADE,
		amount DECIMAL(10, 2) NOT NULL,
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(queryDesigns)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza table ya designs: %v", err)
	}

	_, err = db.Exec(queryOrders)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza table ya orders: %v", err)
	}

	fmt.Println("Tables za Database zimehakikishwa na ziko tayari!")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Imeshindikana kupakia ukurasa: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func designsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, title, description, price, image_url, category FROM designs ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Imeshindikana kusoma designs"}`))
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.Category); err != nil {
			continue
		}
		designs = append(designs, d)
	}

	if designs == nil {
		w.Write([]byte(`[]`))
		return
	}

	json.NewEncoder(w).Encode(designs)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	priceStr := r.FormValue("price")
	imageURL := r.FormValue("image_url")
	category := r.FormValue("category")

	var price float64
	fmt.Sscanf(priceStr, "%f", &price)

	_, err := db.Exec("INSERT INTO designs (title, description, price, image_url, category) VALUES ($1, $2, $3, $4, $5)",
		title, description, price, imageURL, category)
	if err != nil {
		http.Error(w, "Imeshindikana kuhifadhi design: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
