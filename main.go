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

type Design struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	ImageURL    string  `json:"image_url"`
	Category    string  `json:"category"`
	Designer    string  `json:"designer_name"`
	Status      string  `json:"status"`
}

type Order struct {
	ID            int     `json:"id"`
	DesignID      int     `json:"design_id"`
	Phone         string  `json:"phone"`
	Amount        float64 `json:"amount"`
	PaymentStatus string  `json:"payment_status"`
	CreatedAt     string  `json:"created_at"`
}

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

func main() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:password@localhost:5432/sokosmart?sslmode=disable"
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Imeshindikana kuunganisha na Database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Database haipatikani: %v", err)
	}

	initDB()

	// Web Routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/signup", signupHandler)
	http.HandleFunc("/api/signin", signinHandler)
	http.HandleFunc("/api/designs", getDesignsHandler)
	http.HandleFunc("/api/upload", uploadDesignHandler)
	http.HandleFunc("/api/buy", buyDesignHandler)

	// Admin Routes
	http.HandleFunc("/api/admin/login", adminLoginHandler)
	http.HandleFunc("/api/admin/designs", adminGetDesignsHandler)
	http.HandleFunc("/api/admin/approve", adminApproveDesignHandler)
	http.HandleFunc("/api/admin/orders", adminGetOrdersHandler)
	http.HandleFunc("/api/admin/users", adminUsersHandler)
	http.HandleFunc("/api/admin/delete-user", adminDeleteUserHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Seva inaanza kusikiliza kwenye bandari (port) %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() {
	queryUsers := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT DEFAULT 'user',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(queryUsers)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la users: %v", err)
	}

	queryDesigns := `
	CREATE TABLE IF NOT EXISTS designs (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		price NUMERIC NOT NULL,
		image_url TEXT NOT NULL,
		category TEXT NOT NULL,
		designer_name TEXT NOT NULL,
		status TEXT DEFAULT 'pending'
	);`
	_, err = db.Exec(queryDesigns)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la designs: %v", err)
	}

	queryOrders := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		design_id INT NOT NULL,
		phone TEXT NOT NULL,
		amount NUMERIC NOT NULL,
		payment_status TEXT DEFAULT 'pending_tigo_lipa',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(queryOrders)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la orders: %v", err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kusoma taarifa za usajili"})
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	w.Header().Set("Content-Type", "application/json")

	if username == "" || password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina la mtumiaji na nenosiri!"})
		return
	}

	// Angalia kwanza kama jina lipo tayari kabla ya kuingiza
	var existingID int
	err = db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&existingID)
	if err == nil {
		// Ikiwa halina error maanake jina LIPO tayari
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina hili la mtumiaji linatumika tayari! Tafadhali tumia jina lingine."})
		return
	}

	// Kama halipo, ingiza mtumiaji mpya
	_, err = db.Exec("INSERT INTO users (username, password, role) VALUES ($1, $2, 'user')", username, password)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Hitilafu imetokea kwenye database: " + err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Akaunti imefunguliwa kikamilifu! Sasa unaweza kuingia."})
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	w.Header().Set("Content-Type", "application/json")

	var storedPass string
	err := db.QueryRow("SELECT password FROM users WHERE username = $1", username).Scan(&storedPass)
	if err != nil || storedPass != password {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la mtumiaji au nenosiri si sahihi!"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Umeingia kwa mafanikio!"})
}

func getDesignsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, price, image_url, category, designer_name, status FROM designs WHERE status = 'approved'")
	if err != nil {
		http.Error(w, "Imeshindikana kusoma bidhaa", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.Category, &d.Designer, &d.Status); err != nil {
			continue
		}
		designs = append(designs, d)
	}

	if designs == nil {
		designs = []Design{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(designs)
}

func uploadDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
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
		http.Error(w, "Imeshindikana kuweka bidhaa", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func buyDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	designID := r.FormValue("design_id")
	phone := r.FormValue("phone")
	amountStr := r.FormValue("amount")

	var amount float64
	fmt.Sscanf(amountStr, "%f", &amount)

	_, err := db.Exec("INSERT INTO orders (design_id, phone, amount, payment_status) VALUES ($1, $2, $3, 'pending_tigo_lipa')", designID, phone, amount)
	if err != nil {
		http.Error(w, "Imeshindikana kuweka oda", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Oda imepokelewa! Tafadhali kamalisha malipo kupitia Tigo Lipa Namba 45416553."})
}

func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	password := r.FormValue("password")
	
	w.Header().Set("Content-Type", "application/json")
	if password == "khalidsec2026" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false})
	}
}

func adminGetDesignsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, price, image_url, category, designer_name, status FROM designs ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.Category, &d.Designer, &d.Status); err != nil {
			continue
		}
		designs = append(designs, d)
	}

	if designs == nil {
		designs = []Design{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(designs)
}

func adminApproveDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	_, err := db.Exec("UPDATE designs SET status = 'approved' WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Imeshindikana kuidhinisha", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func adminGetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, design_id, phone, amount, payment_status, created_at FROM orders ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
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
		orders = []Order{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func adminUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, username, role, created_at FROM users ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Imeshindikana kusoma watumiaji"}`))
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}

	if users == nil {
		w.Write([]byte(`[]`))
		return
	}

	json.NewEncoder(w).Encode(users)
}

func adminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		http.Error(w, "ID ya mtumiaji haipatikani", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Imeshindikana kufuta mtumiaji", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true, "message": "Mtumiaji amefutwa kabisa kwenye mfumo (Banned & Deleted)!"}`))
}
