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
	ID           int     `json:"id"`
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	ImageURL     string  `json:"image_url"`
	Category     string  `json:"category"`
	DesignerName string  `json:"designer_name"`
	Status       string  `json:"status"`
}

type Order struct {
	ID            int     `json:"id"`
	DesignID      int     `json:"design_id"`
	Phone         string  `json:"phone"`
	Amount        float64 `json:"amount"`
	PaymentStatus string  `json:"payment_status"`
	CreatedAt     string  `json:"created_at"`
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

	initDatabase()

	// Routes za Mfumo
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/designs", designsHandler)
	http.HandleFunc("/api/upload", uploadHandler)
	http.HandleFunc("/api/admin/login", adminLoginHandler)
	http.HandleFunc("/api/admin/designs", adminDesignsHandler)
	http.HandleFunc("/api/admin/orders", adminOrdersHandler)
	http.HandleFunc("/api/admin/approve", approveDesignHandler)
	http.HandleFunc("/api/buy", buyHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Seva inaendelea kusikiliza kwenye port %s...\n", port)
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
		designer_name VARCHAR(100),
		status VARCHAR(20) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	queryOrders := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		design_id INT REFERENCES designs(id) ON DELETE CASCADE,
		phone VARCHAR(20) NOT NULL,
		amount DECIMAL(10, 2) NOT NULL,
		payment_status VARCHAR(20) DEFAULT 'pending',
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

	rows, err := db.Query("SELECT id, title, description, price, image_url, category, designer_name, status FROM designs WHERE status = 'approved' ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Imeshindikana kusoma designs"}`))
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.Category, &d.DesignerName, &d.Status); err != nil {
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
	designerName := r.FormValue("designer_name")

	var price float64
	fmt.Sscanf(priceStr, "%f", &price)

	_, err := db.Exec("INSERT INTO designs (title, description, price, image_url, category, designer_name, status) VALUES ($1, $2, $3, $4, $5, $6, 'pending')",
		title, description, price, imageURL, category, designerName)
	if err != nil {
		http.Error(w, "Imeshindikana kuhifadhi design: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Uthibitisho wa Admin Password (Weka password yako hapa: admin123 au unaweza kuibadilisha unavyotaka)
func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	w.Header().Set("Content-Type", "application/json")

	// Hapa unaweza kubadilisha 'khalidsec2026' kuwa neno lolote la siri unalolitaka
	if password == "khalidsec2026" {
		w.Write([]byte(`{"success": true}`))
	} else {
		w.Write([]byte(`{"success": false}`))
	}
}

func adminDesignsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, title, description, price, image_url, category, designer_name, status FROM designs ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Imeshindikana kusoma orodha"}`))
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.Category, &d.DesignerName, &d.Status); err != nil {
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

func adminOrdersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, design_id, phone, amount, payment_status, created_at FROM orders ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Imeshindikana kusoma oda"}`))
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.DesignID, &o.Phone, &o.Amount, &o.PaymentStatus, &o.CreatedAt); err != nil {
			continue
		}
		orders = append(orders, o)
	}

	if orders == nil {
		w.Write([]byte(`[]`))
		return
	}

	json.NewEncoder(w).Encode(orders)
}

func approveDesignHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID haipatikani", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE designs SET status = 'approved' WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Imeshindikana kusasisha hali", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Imethibitishwa kikamilifu"}`))
}

func buyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	designID := r.FormValue("design_id")
	phone := r.FormValue("phone")
	amountStr := r.FormValue("amount")

	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)

	_, err := db.Exec("INSERT INTO orders (design_id, phone, amount, payment_status) VALUES ($1, $2, $3, 'pending')",
		designID, phone, amount)
	if err != nil {
		http.Error(w, "Imeshindikana kuweka oda ya malipo", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"message": "Oda imepokelewa! Tafadhali fanya malipo kwenda Tigo Lipa Namba 45416553. Admin atakagua na kukutumia bidhaa.",
	}
	json.NewEncoder(w).Encode(response)
}
