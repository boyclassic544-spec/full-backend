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
	ImageURL2       string  `json:"image_url2"`
	ImageURL3       string  `json:"image_url3"`
	ImageURL4       string  `json:"image_url4"`
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
	IDType             string `json:"id_type"`
	IDNumber           string `json:"id_number"`
	IDImageURL         string `json:"id_image_url"`
	RejectionReason    string `json:"rejection_reason"`
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
	http.HandleFunc("/api/profile", profileHandler)
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
	http.HandleFunc("/api/admin/buyers", adminGetBuyersHandler)
	http.HandleFunc("/api/admin/sellers", adminGetSellersHandler)
	http.HandleFunc("/api/admin/approve-user", adminApproveUserHandler)
	http.HandleFunc("/api/admin/reject-user", adminRejectUserHandler)
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
		rejection_reason TEXT DEFAULT '',
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
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS rejection_reason TEXT DEFAULT '';")

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
		status TEXT DEFAULT 'approved',
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
	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS image_url2 TEXT DEFAULT '';")
	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS image_url3 TEXT DEFAULT '';")
	db.Exec("ALTER TABLE designs ADD COLUMN IF NOT EXISTS image_url4 TEXT DEFAULT '';")
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

	w.Header().Set("Content-Type", "application/json")

	var username, password, role, idType, idNumber, rawIDImage string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
			IDType   string `json:"id_type"`
			IDNumber string `json:"id_number"`
			IDImage  string `json:"id_image"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			username = payload.Username
			password = payload.Password
			role = payload.Role
			idType = payload.IDType
			idNumber = payload.IDNumber
			rawIDImage = payload.IDImage
		}
	}

	if username == "" {
		r.ParseMultipartForm(50 << 20)
		r.ParseForm()
		username = r.FormValue("username")
		password = r.FormValue("password")
		role = r.FormValue("role")
		idType = "NIDA"
		idNumber = r.FormValue("nida")
		rawIDImage = r.FormValue("id_card")
	}

	if role == "" {
		role = "buyer"
	}

	if username == "" || password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina la mtumiaji na nenosiri!"})
		return
	}

	var verificationStatus = "approved"
	var idImageURL = ""

	if role == "seller" {
		verificationStatus = "pending"
		if rawIDImage != "" && strings.HasPrefix(rawIDImage, "data:") {
			if savedURL, saveErr := saveBase64Media(rawIDImage); saveErr == nil {
				idImageURL = savedURL
			}
		}
	}

	_, err := db.Exec(`
		INSERT INTO app_accounts (username, password, role, verification_status, id_type, id_number, id_image_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		username, password, role, verificationStatus, idType, idNumber, idImageURL)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina hili la mtumiaji linatumika tayari!"})
		return
	}

	msg := "Akaunti imefunguliwa kikamilifu! Sasa unaweza kuingia."
	if role == "seller" {
		msg = "Akaunti ya muuzaji imefunguliwa! Tafadhali subiri uthibitisho kutoka kwa uongozi."
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"message":             msg,
		"role":                role,
		"verification_status": verificationStatus,
	})
}

func signinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var username, password string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			username = payload.Username
			password = payload.Password
		}
	}

	if username == "" {
		r.ParseForm()
		username = r.FormValue("username")
		password = r.FormValue("password")
	}

	var storedPass, role, verificationStatus, rejectionReason string
	err := db.QueryRow("SELECT password, role, verification_status, COALESCE(rejection_reason, '') FROM app_accounts WHERE username = $1", username).Scan(&storedPass, &role, &verificationStatus, &rejectionReason)
	if err != nil || storedPass != password {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la mtumiaji au nenosiri si sahihi!"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"message":             "Umeingia kwa mafanikio!",
		"username":            username,
		"role":                role,
		"verification_status": verificationStatus,
		"rejection_reason":    rejectionReason,
	})
}

func profileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	username := r.URL.Query().Get("username")
	if username == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Username inahitajika"})
		return
	}

	var u User
	err := db.QueryRow("SELECT id, username, role, verification_status, COALESCE(id_type, ''), COALESCE(id_number, ''), COALESCE(id_image_url, ''), COALESCE(rejection_reason, '') FROM app_accounts WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.IDType, &u.IDNumber, &u.IDImageURL, &u.RejectionReason)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Mtumiaji hajapatikana"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"username":            u.Username,
		"role":                u.Role,
		"verification_status": u.VerificationStatus,
		"id_type":             u.IDType,
		"id_number":           u.IDNumber,
		"id_image_url":        u.IDImageURL,
		"rejection_reason":    u.RejectionReason,
	})
}

func getDesignsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, price, image_url, COALESCE(image_url2,''), COALESCE(image_url3,''), COALESCE(image_url4,''), video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs WHERE status = 'approved' ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana kusoma bidhaa", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.ImageURL2, &d.ImageURL3, &d.ImageURL4, &d.VideoURL, &d.Category, &d.Designer, &d.Location, &d.VendorPhone, &d.Status, &d.RejectionReason); err != nil {
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
	rows, err := db.Query("SELECT id, title, description, price, image_url, COALESCE(image_url2,''), COALESCE(image_url3,''), COALESCE(image_url4,''), video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs WHERE designer_name = $1 ORDER BY id DESC", designerName)
	if err != nil {
		http.Error(w, "Imeshindikana kusoma bidhaa zako", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.ImageURL2, &d.ImageURL3, &d.ImageURL4, &d.VideoURL, &d.Category, &d.Designer, &d.Location, &d.VendorPhone, &d.Status, &d.RejectionReason); err != nil {
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
		ImageURL2    string  `json:"image_url2"`
		ImageURL3    string  `json:"image_url3"`
		ImageURL4    string  `json:"image_url4"`
		VideoURL     string  `json:"video_url"`
		Category     string  `json:"category"`
		DesignerName string  `json:"designer_name"`
		Location     string  `json:"location"`
		VendorPhone  string  `json:"vendor_phone"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 80<<20)
	err := json.NewDecoder(r.Body).Decode(&payload)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Faili ni kubwa sana au kuna tatizo kwenye data! Jaribu kupunguza ukubwa wa video au picha."})
		return
	}

	if payload.DesignerName != "" {
		var vStatus string
		err = db.QueryRow("SELECT verification_status FROM app_accounts WHERE username = $1", payload.DesignerName).Scan(&vStatus)
		if err != nil || vStatus != "approved" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Akaunti yako bado haijapitishwa na Admin au imefutwa. Huwezi kupost bidhaa kwa sasa.",
			})
			return
		}
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Jina la mtengenezaji (designer_name) linahitajika.",
		})
		return
	}

	finalImg := payload.ImageURL
	if strings.HasPrefix(payload.ImageURL, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL); saveErr == nil {
			finalImg = url
		}
	}

	finalImg2 := payload.ImageURL2
	if strings.HasPrefix(payload.ImageURL2, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL2); saveErr == nil {
			finalImg2 = url
		}
	}

	finalImg3 := payload.ImageURL3
	if strings.HasPrefix(payload.ImageURL3, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL3); saveErr == nil {
			finalImg3 = url
		}
	}

	finalImg4 := payload.ImageURL4
	if strings.HasPrefix(payload.ImageURL4, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL4); saveErr == nil {
			finalImg4 = url
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

	_, err = db.Exec("INSERT INTO designs (title, description, price, image_url, image_url2, image_url3, image_url4, video_url, category, designer_name, location, vendor_phone, status, rejection_reason) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'approved', '')",
		payload.Title, payload.Description, payload.Price, finalImg, finalImg2, finalImg3, finalImg4, finalVideo, payload.Category, payload.DesignerName, payload.Location, payload.VendorPhone)

	if err != nil {
		log.Printf("DB error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuweka bidhaa kwenye database"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imechapishwa sokoni kwa mafanikio!"})
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
		ImageURL2   string  `json:"image_url2"`
		ImageURL3   string  `json:"image_url3"`
		ImageURL4   string  `json:"image_url4"`
		VideoURL    string  `json:"video_url"`
		Category    string  `json:"category"`
		Location    string  `json:"location"`
		VendorPhone string  `json:"vendor_phone"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 80<<20)
	err := json.NewDecoder(r.Body).Decode(&payload)

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Faili ni kubwa sana au kuna tatizo kwenye data! Jaribu kupunguza ukubwa wa video au picha."})
		return
	}

	finalImg := payload.ImageURL
	if strings.HasPrefix(payload.ImageURL, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL); saveErr == nil {
			finalImg = url
		}
	}

	finalImg2 := payload.ImageURL2
	if strings.HasPrefix(payload.ImageURL2, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL2); saveErr == nil {
			finalImg2 = url
		}
	}

	finalImg3 := payload.ImageURL3
	if strings.HasPrefix(payload.ImageURL3, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL3); saveErr == nil {
			finalImg3 = url
		}
	}

	finalImg4 := payload.ImageURL4
	if strings.HasPrefix(payload.ImageURL4, "data:") {
		if url, saveErr := saveBase64Media(payload.ImageURL4); saveErr == nil {
			finalImg4 = url
		}
	}

	finalVideo := payload.VideoURL
	if strings.HasPrefix(payload.VideoURL, "data:") {
		if url, saveErr := saveBase64Media(payload.VideoURL); saveErr == nil {
			finalVideo = url
		}
	}

	_, err = db.Exec("UPDATE designs SET title = $1, description = $2, price = $3, image_url = $4, image_url2 = $5, image_url3 = $6, image_url4 = $7, video_url = $8, category = $9, location = $10, vendor_phone = $11, status = 'approved', rejection_reason = '' WHERE id = $12",
		payload.Title, payload.Description, payload.Price, finalImg, finalImg2, finalImg3, finalImg4, finalVideo, payload.Category, payload.Location, payload.VendorPhone, payload.ID)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuhifadhi mabadiliko"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imesasishwa kikamilifu!"})
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
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Oda imepokelewa! Lipa kupitia Tigo Lipa Namba 45416553."})
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
	rows, err := db.Query("SELECT id, title, description, price, image_url, COALESCE(image_url2,''), COALESCE(image_url3,''), COALESCE(image_url4,''), video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var designs []Design
	for rows.Next() {
		var d Design
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Price, &d.ImageURL, &d.ImageURL2, &d.ImageURL3, &d.ImageURL4, &d.VideoURL, &d.Category, &d.Designer, &d.Location, &d.VendorPhone, &d.Status, &d.RejectionReason); err != nil {
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

	w.Header().Set("Content-Type", "application/json")
	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "ID ya bidhaa inahitajika"})
		return
	}

	_, err := db.Exec("UPDATE designs SET status = 'approved', rejection_reason = '' WHERE id = $1", id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuidhinisha bidhaa"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imeidhinishwa kikamilifu!"})
}

func adminRejectDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	w.Header().Set("Content-Type", "application/json")
	id := r.URL.Query().Get("id")
	reason := r.FormValue("reason")
	if reason == "" {
		reason = "Haikutimiza vigezo."
	}

	_, err := db.Exec("UPDATE designs SET status = 'rejected', rejection_reason = $1 WHERE id = $2", reason, id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kukataa bidhaa"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imekataliwa!"})
}

func adminDeleteDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	designID := r.URL.Query().Get("id")
	_, err := db.Exec("DELETE FROM designs WHERE id = $1", designID)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imefutwa!"})
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

	rows, err := db.Query("SELECT id, username, role, verification_status, COALESCE(id_type, ''), COALESCE(id_number, ''), COALESCE(id_image_url, ''), COALESCE(rejection_reason, ''), created_at FROM app_accounts ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Imeshindikana kusoma watumiaji"}`))
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.IDType, &u.IDNumber, &u.IDImageURL, &u.RejectionReason, &u.CreatedAt); err != nil {
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

func adminGetBuyersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.Query("SELECT id, username, role, verification_status, created_at FROM app_accounts WHERE role = 'buyer' ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`[]`))
		return
	}
	defer rows.Close()

	var buyers []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.CreatedAt); err != nil {
			continue
		}
		buyers = append(buyers, u)
	}

	if buyers == nil {
		buyers = []User{}
	}

	json.NewEncoder(w).Encode(buyers)
}

func adminGetSellersHandler(w http.ResponseWriter, r *http.SourceContextRequest) { // kept standard standard signature below
// standard signature:
}

func adminGetSellersHandlerReal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.Query("SELECT id, username, role, verification_status, COALESCE(id_type, ''), COALESCE(id_number, ''), COALESCE(id_image_url, ''), COALESCE(rejection_reason, ''), created_at FROM app_accounts WHERE role = 'seller' ORDER BY id DESC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`[]`))
		return
	}
	defer rows.Close()

	var sellers []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.IDType, &u.IDNumber, &u.IDImageURL, &u.RejectionReason, &u.CreatedAt); err != nil {
			continue
		}
		sellers = append(sellers, u)
	}

	if sellers == nil {
		sellers = []User{}
	}

	json.NewEncoder(w).Encode(sellers)
}

// Re-assign to match exact original function name used in main() routing
func adminGetSellersHandler(w http.ResponseWriter, r *http.Request) {
	adminGetSellersHandlerReal(w, r)
}

func adminApproveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	userID := r.URL.Query().Get("id")
	if userID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "ID ya mtumiaji inahitajika"})
		return
	}

	_, err := db.Exec("UPDATE app_accounts SET verification_status = 'approved', rejection_reason = '' WHERE id = $1", userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuidhinisha"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Muuzaji amepitishwa kikamilifu!"})
}

func adminRejectUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	w.Header().Set("Content-Type", "application/json")
	userID := r.URL.Query().Get("id")
	reason := r.FormValue("reason")
	if reason == "" {
		reason = "Akaunti imekataliwa na Msimamizi."
	}

	_, err := db.Exec("UPDATE app_accounts SET verification_status = 'rejected', rejection_reason = $1 WHERE id = $2", reason, userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kukataa akaunti"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Akaunti ya muuzaji imekataliwa na ujumbe umehifadhiwa!"})
}

func adminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	_, err := db.Exec("DELETE FROM app_accounts WHERE id = $1", userID)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta mtumiaji"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Mtumiaji amefutwa kabisa!"})
}
