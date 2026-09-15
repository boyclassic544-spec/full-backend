package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type Design struct {
	ID              int     `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	ImageURL        string  `json:"image_url"`
	VideoURL        string  `json:"video_url"`
	Category        string  `json:"category"`
	Designer        string  `json:"designer_name"`
	Location        string  `json:"location"`
	VendorPhone     string  `json:"vendor_phone"`
	Status          string  `json:"status"`
	RejectionReason string  `json:"rejection_reason"`
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
	ID                 int    `json:"id"`
	Username           string `json:"username"`
	Role               string `json:"role"`
	VerificationStatus string `json:"verification_status"`
	CreatedAt          string `json:"created_at"`
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

	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/signup", signupHandler)
	http.HandleFunc("/api/signin", signinHandler)
	http.HandleFunc("/api/designs", getDesignsHandler)
	http.HandleFunc("/api/my-designs", getMyDesignsHandler)
	http.HandleFunc("/api/upload", uploadDesignJSONHandler)
	http.HandleFunc("/api/update-design", updateDesignJSONHandler)
	http.HandleFunc("/api/delete-design", deleteMyDesignHandler)
	http.HandleFunc("/api/buy", buyDesignHandler)

	http.HandleFunc("/api/admin/login", adminLoginHandler)
	http.HandleFunc("/api/admin/designs", adminGetDesignsHandler)
	http.HandleFunc("/api/admin/approve", adminApproveDesignHandler)
	http.HandleFunc("/api/admin/reject", adminRejectDesignHandler)
	http.HandleFunc("/api/admin/delete-design", adminDeleteDesignHandler)
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
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		log.Fatalf("Imeshindikana kutengeneza folder la uploads: %v", err)
	}

	queryUsers := `
	CREATE TABLE IF NOT EXISTS app_accounts (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		role TEXT DEFAULT 'buyer',
		verification_status TEXT DEFAULT 'approved',
		id_type TEXT DEFAULT '',
		id_number TEXT DEFAULT '',
		id_image_url TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(queryUsers)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la app_accounts: %v", err)
	}

	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'buyer';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS verification_status TEXT DEFAULT 'approved';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS id_type TEXT DEFAULT '';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS id_number TEXT DEFAULT '';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS id_image_url TEXT DEFAULT '';")

	queryDesigns := `
	CREATE TABLE IF NOT EXISTS designs (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		price NUMERIC NOT NULL,
		image_url TEXT NOT NULL,
		video_url TEXT DEFAULT '',
		category TEXT NOT NULL,
		designer_name TEXT DEFAULT '',
		location TEXT DEFAULT 'Tanzania',
		vendor_phone TEXT DEFAULT '',
		status TEXT DEFAULT 'pending',
		rejection_reason TEXT DEFAULT ''
	);`
	_, err = db.Exec(queryDesigns)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la designs: %v", err)
	}

	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS designer_name TEXT DEFAULT '';")
	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS location TEXT DEFAULT 'Tanzania';")
	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS vendor_phone TEXT DEFAULT '';")
	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS rejection_reason TEXT DEFAULT '';")
	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS video_url TEXT DEFAULT '';")
	db.Exec("ALTER TABLE designs ALTER COLUMN image_url TYPE TEXT;")

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

func saveBase64Media(dataURL string) (string, error) {
	parts := strings.SplitN(dataURL, ";base64,", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("muundo wa media si sahihi")
	}

	meta := parts[0]
	ext := ".jpg"
	
	if strings.Contains(meta, "image/png") {
		ext = ".png"
	} else if strings.Contains(meta, "image/webp") {
		ext = ".webp"
	} else if strings.Contains(meta, "image/gif") {
		ext = ".gif"
	} else if strings.Contains(meta, "video/mp4") {
		ext = ".mp4"
	} else if strings.Contains(meta, "video/webm") {
		ext = ".webm"
	}

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), rand.Intn(100000), ext)
	filePath := filepath.Join("./uploads", filename)

	if err := os.WriteFile(filePath, decoded, 0644); err != nil {
		return "", err
	}

	return "/uploads/" + filename, nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	w.Header().Set("Content-Type", "application/json")

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
		IDType   string `json:"id_type"`
		IDNumber string `json:"id_number"`
		IDImage  string `json:"id_image"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kusoma taarifa au faili ni kubwa sana"})
		return
	}

	username := payload.Username
	password := payload.Password
	role := payload.Role
	if role == "" {
		role = "buyer"
	}

	if username == "" || password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina la mtumiaji na nenosiri!"})
		return
	}

	var verificationStatus = "approved"
	var idType = ""
	var idNumber = ""
	var idImageURL = ""

	if role == "seller" {
		verificationStatus = "pending"
		idType = payload.IDType
		idNumber = payload.IDNumber
		rawIDImage := payload.IDImage

		if rawIDImage != "" && strings.HasPrefix(rawIDImage, "data:") {
			if savedURL, saveErr := saveBase64Media(rawIDImage); saveErr == nil {
				idImageURL = savedURL
			}
		}
	}

	_, err = db.Exec(`
		INSERT INTO app_accounts (username, password, role, verification_status, id_type, id_number, id_image_url) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		username, password, role, verificationStatus, idType, idNumber, idImageURL)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina hili la mtumiaji linatumika tayari!"})
		return
	}

	msg := "Akaunti imefunguliwa kikamilifu! Sasa unaweza kuingia."
	if role == "seller" {
		msg = "Akaunti ya muuzaji imefunguliwa! Tafadhali subiri uthibitisho kutoka kwa uongozi kabla ya kuweka bidhaa."
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, 
		"message": msg,
		"role": role,
		"verification_status": verificationStatus,
	})
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kusoma taarifa za kuingia"})
		return
	}

	var storedPass, role, verificationStatus string
	err = db.QueryRow("SELECT password, role, verification_status FROM app_accounts WHERE username = $1", payload.Username).Scan(&storedPass, &role, &verificationStatus)
	if err != nil || storedPass != payload.Password {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la mtumiaji au nenosiri si sahihi!"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, 
		"message": "Umeingia kwa mafanikio!",
		"username": payload.Username,
		"role": role,
		"verification_status": verificationStatus,
	})
}

func getDesignsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, price, image_url, video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs WHERE status = 'approved' ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana kusoma bidhaa", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.VideoURL, &d.Category, &d.Designer, &d.Location, &d.VendorPhone, &d.Status, &d.RejectionReason); err != nil {
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

func getMyDesignsHandler(w http.ResponseWriter, r *http.Request) {
	designerName := r.URL.Query().Get("designer")
	rows, err := db.Query("SELECT id, title, description, price, image_url, video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs WHERE designer_name = $1 ORDER BY id DESC", designerName)
	if err != nil {
		http.Error(w, "Imeshindikana kusoma bidhaa zako", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.VideoURL, &d.Category, &d.Designer, &d.Location, &d.VendorPhone, &d.Status, &d.RejectionReason); err != nil {
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

func uploadDesignJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Title        string  `json:"title"`
		Description  string  `json:"description"`
		Price        float64 `json:"price"`
		ImageURL     string  `json:"image_url"`
		VideoURL     string  `json:"video_url"`
		Category     string  `json:"category"`
		DesignerName string  `json:"designer_name"`
		Location     string  `json:"location"`
		VendorPhone  string  `json:"vendor_phone"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	err := json.NewDecoder(r.Body).Decode(&payload)
	
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Faili ni kubwa sana au kuna tatizo kwenye data!"})
		return
	}

	if payload.DesignerName != "" {
		var vStatus string
		err = db.QueryRow("SELECT verification_status FROM app_accounts WHERE username = $1", payload.DesignerName).Scan(&vStatus)
		if err == nil && vStatus != "approved" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, 
				"message": "Akaunti yako haijathibitishwa na Admin bado. Huwezi kupost bidhaa kwa sasa.",
			})
			return
		}
	}

	finalImg := payload.ImageURL
	if strings.HasPrefix(payload.ImageURL, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL); saveErr == nil {
			finalImg = url
		}
	}

	finalVideo := payload.VideoURL
	if strings.HasPrefix(payload.VideoURL, "data:") {
		if url, saveErr := saveBase64Media(payload.VideoURL); saveErr == nil {
			finalVideo = url
		}
	}

	if finalImg == "" {
		finalImg = "https://via.placeholder.com/300"
	}
	if payload.Location == "" {
		payload.Location = "Morogoro, Tanzania"
	}

	_, err = db.Exec("INSERT INTO designs (title, description, price, image_url, video_url, category, designer_name, location, vendor_phone, status, rejection_reason) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending', '')",
		payload.Title, payload.Description, payload.Price, finalImg, finalVideo, payload.Category, payload.DesignerName, payload.Location, payload.VendorPhone)
	
	if err != nil {
		log.Printf("DB error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuweka bidhaa kwenye database"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa yako (pamoja na picha na video) imewasilishwa kwa ukaguzi!"})
}

func updateDesignJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		ID          int     `json:"id"`
		Title       string  `json:"title"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		ImageURL    string  `json:"image_url"`
		VideoURL    string  `json:"video_url"`
		Category    string  `json:"category"`
		Location    string  `json:"location"`
		VendorPhone string  `json:"vendor_phone"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 25<<20)
	err := json.NewDecoder(r.Body).Decode(&payload)
	
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kusoma taarifa za marekebisho"})
		return
	}

	finalImg := payload.ImageURL
	if strings.HasPrefix(payload.ImageURL, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL); saveErr == nil {
			finalImg = url
		}
	}

	finalVideo := payload.VideoURL
	if strings.HasPrefix(payload.VideoURL, "data:") {
		if url, saveErr := saveBase64Media(payload.VideoURL); saveErr == nil {
			finalVideo = url
		}
	}

	_, err = db.Exec("UPDATE designs SET title = $1, description = $2, price = $3, image_url = $4, video_url = $5, category = $6, location = $7, vendor_phone = $8, status = 'pending', rejection_reason = '' WHERE id = $9",
		payload.Title, payload.Description, payload.Price, finalImg, finalVideo, payload.Category, payload.Location, payload.VendorPhone, payload.ID)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuhifadhi mabadiliko"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imesasishwa na kurudishwa kwenye ukaguzi!"})
}

func deleteMyDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	designID := r.URL.Query().Get("id")
	_, err := db.Exec("DELETE FROM designs WHERE id = $1", designID)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta bidhaa"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imefutwa!"})
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
	rows, err := db.Query("SELECT id, title, description, price, image_url, video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.VideoURL, &d.Category, &d.Designer, &d.Location, &d.VendorPhone, &d.Status, &d.RejectionReason); err != nil {
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
	_, err := db.Exec("UPDATE designs SET status = 'approved', rejection_reason = '' WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Imeshindikana kuidhinisha", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func adminRejectDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Taarifa mbovu", http.StatusBadRequest)
		return
	}

	id := r.URL.Query().Get("id")
	reason := r.FormValue("reason")
	if reason == "" {
		reason = "Haikutimiza vigezo vya ubora."
	}

	_, err = db.Exec("UPDATE designs SET status = 'rejected', rejection_reason = $1 WHERE id = $2", reason, id)
	if err != nil {
		http.Error(w, "Imeshindikana kukataa bidhaa", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imekataliwa kikamilifu."})
}

func adminDeleteDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	designID := r.URL.Query().Get("id")
	if designID == "" {
		http.Error(w, "ID haipatikani", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM designs WHERE id = $1", designID)
	if err != nil {
		http.Error(w, "Imeshindikana kufuta bidhaa", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imefutwa kabisa na Admin!"})
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

	rows, err := db.Query("SELECT id, username, role, verification_status, created_at FROM app_accounts ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Imeshindikana kusoma watumiaji"}`))
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.CreatedAt); err != nil {
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
		http.Error(w, "ID haipatikani", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM app_accounts WHERE id = $1", userID)
	if err != nil {
		http.Error(w, "Imeshindikana kufuta mtumiaji", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Mtumiaji amefutwa kabisa!"})
}
