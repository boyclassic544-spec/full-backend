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

type Story struct {
	ID              int    `json:"id"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	CoverImage      string `json:"cover_image"`
	StorytellerName string `json:"storyteller_name"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason"`
	CreatedAt       string `json:"created_at"`
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
	SubscriptionStatus string `json:"subscription_status"`
	TrialEndsAt        string `json:"trial_ends_at"`
	PaymentPhone       string `json:"payment_phone"`
	PaymentName        string `json:"payment_name"`
	CreatedAt          string `json:"created_at"`
}

type Feedback struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Message   string `json:"message"`
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

	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads"))))

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/signup", signupHandler)
	http.HandleFunc("/api/signin", signinHandler)
	http.HandleFunc("/api/profile", profileHandler)
	http.HandleFunc("/api/submit-subscription-payment", submitSubscriptionPaymentHandler)
	http.HandleFunc("/api/feedback", feedbackHandler)
	
	// Routes za Soko Kuu (Sellers / Products)
	http.HandleFunc("/api/designs", getDesignsHandler)
	http.HandleFunc("/api/my-designs", getMyDesignsHandler)
	http.HandleFunc("/api/upload", uploadDesignJSONHandler)
	http.HandleFunc("/api/update-design", updateDesignJSONHandler)
	http.HandleFunc("/api/delete-design", deleteMyDesignHandler)
	http.HandleFunc("/api/buy", buyDesignHandler)

	// Routes Maalum za Hadithi (Stories Feed & Storyteller Dashboard)
	http.HandleFunc("/api/stories", getStoriesHandler)
	http.HandleFunc("/api/my-stories", getMyStoriesHandler)
	http.HandleFunc("/api/storyteller/upload", uploadStoryJSONHandler)
	http.HandleFunc("/api/delete-story", deleteMyStoryHandler)

	// Admin Routes
	http.HandleFunc("/api/admin/login", adminLoginHandler)
	http.HandleFunc("/api/admin/designs", adminGetDesignsHandler)
	http.HandleFunc("/api/admin/approve", adminApproveDesignHandler)
	http.HandleFunc("/api/admin/reject", adminRejectDesignHandler)
	http.HandleFunc("/api/admin/delete-design", adminDeleteDesignHandler)
	
	http.HandleFunc("/api/admin/stories", adminGetStoriesHandler)
	http.HandleFunc("/api/admin/approve-story", adminApproveStoryHandler)
	http.HandleFunc("/api/admin/reject-story", adminRejectStoryHandler)
	http.HandleFunc("/api/admin/delete-story", adminDeleteStoryHandler)

	http.HandleFunc("/api/admin/orders", adminGetOrdersHandler)
	http.HandleFunc("/api/admin/users", adminUsersHandler)
	http.HandleFunc("/api/admin/buyers", adminGetBuyersHandler)
	http.HandleFunc("/api/admin/sellers", adminGetSellersHandler)
	http.HandleFunc("/api/admin/storytellers", adminGetStorytellersHandler)
	http.HandleFunc("/api/admin/approve-user", adminApproveUserHandler)
	http.HandleFunc("/api/admin/reject-user", adminRejectUserHandler)
	http.HandleFunc("/api/admin/delete-user", adminDeleteUserHandler)
	http.HandleFunc("/api/admin/approve-subscription", adminApproveSubscriptionHandler)
	http.HandleFunc("/api/admin/add-subadmin", createSubAdminHandler)
	http.HandleFunc("/api/admin/feedback", adminGetFeedbackHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	fmt.Printf("Seva inaanza kusikiliza kwenye bandari (port) %s...\n", port)
	log.Fatal(srv.ListenAndServe())
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
		subscription_status TEXT DEFAULT 'free',
		trial_ends_at TIMESTAMP,
		payment_phone TEXT DEFAULT '',
		payment_name TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(queryUsers)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la app_accounts: %v", err)
	}

	queryDesigns := `
	CREATE TABLE IF NOT EXISTS designs (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		price NUMERIC DEFAULT 0,
		image_url TEXT NOT NULL,
		image_url2 TEXT DEFAULT '',
		image_url3 TEXT DEFAULT '',
		image_url4 TEXT DEFAULT '',
		video_url TEXT DEFAULT '',
		category TEXT DEFAULT 'Bidhaa',
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

	queryStories := `
	CREATE TABLE IF NOT EXISTS stories (
		id SERIAL PRIMARY KEY,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		cover_image TEXT DEFAULT '',
		storyteller_name TEXT NOT NULL,
		status TEXT DEFAULT 'approved',
		rejection_reason TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(queryStories)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la stories: %v", err)
	}

	queryOrders := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		design_id INT NOT NULL,
		phone TEXT NOT NULL,
		amount NUMERIC DEFAULT 0,
		payment_status TEXT DEFAULT 'pending_tigo_lipa',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(queryOrders)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la orders: %v", err)
	}

	queryFeedback := `
	CREATE TABLE IF NOT EXISTS feedback (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT DEFAULT '',
		message TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(queryFeedback)
	if err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la feedback: %v", err)
	}

	db.Exec("ALTER TABLE stories ADD COLUMN IF NOT EXISTS cover_image TEXT DEFAULT '';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS subscription_status TEXT DEFAULT 'free';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP;")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS payment_phone TEXT DEFAULT '';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS payment_name TEXT DEFAULT '';")
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
		idType = r.FormValue("id_type")
		if idType == "" {
			idType = "NIDA"
		}
		idNumber = r.FormValue("nida")
		if idNumber == "" {
			idNumber = r.FormValue("id_number")
		}
		rawIDImage = r.FormValue("id_card")
		if rawIDImage == "" {
			rawIDImage = r.FormValue("id_image")
		}
	}

	if role == "" {
		role = "buyer"
	}

	var verificationStatus = "approved"
	if role == "seller" || role == "storyteller" || role == "stela" {
		verificationStatus = "pending"
		if role == "stela" {
			role = "storyteller"
		}
	}

	var idImageURL = ""
	if rawIDImage != "" && strings.HasPrefix(rawIDImage, "data:") {
		if savedURL, saveErr := saveBase64Media(rawIDImage); saveErr == nil {
			idImageURL = savedURL
		}
	}

	if username == "" || password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina la mtumiaji na nenosiri!"})
		return
	}

	trialEnds := time.Now().Add(30 * 24 * time.Hour)
	subStatus := "free"
	if role == "buyer" {
		subStatus = "active"
	}

	_, err := db.Exec(`
		INSERT INTO app_accounts (username, password, role, verification_status, id_type, id_number, id_image_url, subscription_status, trial_ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		username, password, role, verificationStatus, idType, idNumber, idImageURL, subStatus, trialEnds)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina hili la mtumiaji linatumika tayari!"})
		return
	}

	msg := "Akaunti imefunguliwa kikamilifu! Sasa unaweza kuingia."
	if role == "seller" || role == "storyteller" {
		msg = "Akaunti imefunguliwa! Una mwezi 1 wa bure (Free Plan). Mwezi ujao utachangia Tsh 5,000 kupitia Tigo Lipa (45416553 - SALMIN TAMIMU HUSEIN). Tafadhali subiri uthibitisho wa Admin."
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

	var storedPass, role, verificationStatus, rejectionReason, subStatus string
	var trialEnds sql.NullTime
	err := db.QueryRow("SELECT password, role, verification_status, COALESCE(rejection_reason, ''), COALESCE(subscription_status, 'free'), trial_ends_at FROM app_accounts WHERE username = $1", username).Scan(&storedPass, &role, &verificationStatus, &rejectionReason, &subStatus, &trialEnds)
	if err != nil || storedPass != password {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la mtumiaji au nenosiri si sahihi!"})
		return
	}

	isExpired := false
	if role != "buyer" && role != "sub_admin" && trialEnds.Valid && time.Now().After(trialEnds.Time) && subStatus != "active" {
		isExpired = true
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"message":             "Umeingia kwa mafanikio!",
		"username":            username,
		"role":                role,
		"verification_status": verificationStatus,
		"rejection_reason":    rejectionReason,
		"subscription_status": subStatus,
		"is_expired":          isExpired,
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
	var trialEnds sql.NullTime
	err := db.QueryRow("SELECT id, username, role, verification_status, COALESCE(id_type, ''), COALESCE(id_number, ''), COALESCE(id_image_url, ''), COALESCE(rejection_reason, ''), COALESCE(subscription_status, 'free'), trial_ends_at, COALESCE(payment_phone, ''), COALESCE(payment_name, '') FROM app_accounts WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.IDType, &u.IDNumber, &u.IDImageURL, &u.RejectionReason, &u.SubscriptionStatus, &trialEnds, &u.PaymentPhone, &u.PaymentName)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Mtumiaji hajapatikana"})
		return
	}

	if trialEnds.Valid {
		u.TrialEndsAt = trialEnds.Time.Format("2006-01-02 15:04:05")
	}

	isExpired := false
	if u.Role != "buyer" && u.Role != "sub_admin" && trialEnds.Valid && time.Now().After(trialEnds.Time) && u.SubscriptionStatus != "active" {
		isExpired = true
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
		"subscription_status": u.SubscriptionStatus,
		"trial_ends_at":       u.TrialEndsAt,
		"is_expired":          isExpired,
		"payment_phone":       u.PaymentPhone,
		"payment_name":        u.PaymentName,
	})
}

func submitSubscriptionPaymentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var username, paymentPhone, paymentName string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Username     string `json:"username"`
			PaymentPhone string `json:"payment_phone"`
			PaymentName  string `json:"payment_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			username = payload.Username
			paymentPhone = payload.PaymentPhone
			paymentName = payload.PaymentName
		}
	}

	if username == "" {
		r.ParseForm()
		username = r.FormValue("username")
		paymentPhone = r.FormValue("payment_phone")
		paymentName = r.FormValue("payment_name")
	}

	if username == "" || paymentPhone == "" || paymentName == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Tafadhali jaza namba ya simu na jina ulilofanyia malipo!"})
		return
	}

	_, err := db.Exec("UPDATE app_accounts SET payment_phone = $1, payment_name = $2, subscription_status = 'pending_payment' WHERE username = $3", paymentPhone, paymentName, username)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kutuma taarifa za malipo"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Ombi lako la malipo limetumwa kwa Admin! Tafadhali subiri uhakiki.",
	})
}

func adminApproveSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	userID := r.URL.Query().Get("id")
	if userID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "ID inahitajika"})
		return
	}

	var currentTrial sql.NullTime
	err := db.QueryRow("SELECT trial_ends_at FROM app_accounts WHERE id = $1", userID).Scan(&currentTrial)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Mtumiaji hajapatikana"})
		return
	}

	baseTime := time.Now()
	if currentTrial.Valid && currentTrial.Time.After(baseTime) {
		baseTime = currentTrial.Time
	}
	newTrial := baseTime.Add(30 * 24 * time.Hour)

	_, err = db.Exec("UPDATE app_accounts SET subscription_status = 'active', trial_ends_at = $1 WHERE id = $2", newTrial, userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kusasisha usajili"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Hongera! Malipo yamethibitishwa na akaunti imeongezewa mwezi 1 mpya kikamilifu.",
	})
}

func feedbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	var name, email, message string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Name    string `json:"name"`
			Email   string `json:"email"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			name = payload.Name
			email = payload.Email
			message = payload.Message
		}
	}

	if name == "" {
		r.ParseForm()
		name = r.FormValue("name")
		email = r.FormValue("email")
		message = r.FormValue("message")
	}

	if name == "" || message == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Tafadhali jaza jina lako na ujumbe wako!"})
		return
	}

	_, err := db.Exec("INSERT INTO feedback (name, email, message) VALUES ($1, $2, $3)", name, email, message)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kutuma maoni."})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Asante sana! Maoni yako yamewasilishwa kikamilifu.",
	})
}

func checkSubscriptionAndVerification(username string) (bool, string) {
	var role, vStatus, subStatus string
	var trialEnds sql.NullTime

	err := db.QueryRow("SELECT role, verification_status, COALESCE(subscription_status, 'free'), trial_ends_at FROM app_accounts WHERE username = $1", username).
		Scan(&role, &vStatus, &subStatus, &trialEnds)

	if err != nil {
		return false, "Mtumiaji hajapatikana"
	}

	if vStatus != "approved" {
		return false, "Akaunti yako bado haijapitishwa na Admin."
	}

	if role == "buyer" || role == "sub_admin" {
		return true, ""
	}

	if trialEnds.Valid && time.Now().After(trialEnds.Time) && subStatus != "active" {
		return false, "Mwezi wako wa bure umekwisha. Tafadhali lipa Tsh 5,000 kwenda Tigo Lipa: 45416553 (SALMIN TAMIMU HUSEIN) kisha utume namba yako na jina ili Admin akujulishe."
	}

	return true, ""
}

func getDesignsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, price, image_url, COALESCE(image_url2,''), COALESCE(image_url3,''), COALESCE(image_url4,''), video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs WHERE status = 'approved' ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana kusoma", http.StatusInternalServerError)
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

func uploadDesignJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var title, description, category, designerName, location, vendorPhone, videoURL string
	var img1, img2, img3, img4 string
	var price float64

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
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
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			title = payload.Title
			description = payload.Description
			price = payload.Price
			img1 = payload.ImageURL
			img2 = payload.ImageURL2
			img3 = payload.ImageURL3
			img4 = payload.ImageURL4
			videoURL = payload.VideoURL
			category = payload.Category
			designerName = payload.DesignerName
			location = payload.Location
			vendorPhone = payload.VendorPhone
		}
	}

	if title == "" {
		r.Body = http.MaxBytesReader(w, r.Body, 80<<20)
		r.ParseMultipartForm(80 << 20)
		r.ParseForm()
		title = r.FormValue("title")
		description = r.FormValue("description")
		category = r.FormValue("category")
		
		designerName = r.FormValue("designer_name")
		if designerName == "" {
			designerName = r.FormValue("designer")
		}
		
		location = r.FormValue("location")
		vendorPhone = r.FormValue("vendor_phone")
		videoURL = r.FormValue("video_url")
		
		priceStr := r.FormValue("price")
		fmt.Sscanf(priceStr, "%f", &price)

		img1 = r.FormValue("image_url")
		if img1 == "" {
			img1 = r.FormValue("image")
		}
		img2 = r.FormValue("image_url2")
		img3 = r.FormValue("image_url3")
		img4 = r.FormValue("image_url4")
	}

	if designerName != "" {
		if ok, errMsg := checkSubscriptionAndVerification(designerName); !ok {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": errMsg,
			})
			return
		}
	}

	if strings.HasPrefix(img1, "data:") {
		if url, saveErr := saveBase64Media(img1); saveErr == nil {
			img1 = url
		}
	}
	if strings.HasPrefix(img2, "data:") {
		if url, saveErr := saveBase64Media(img2); saveErr == nil {
			img2 = url
		}
	}
	if strings.HasPrefix(img3, "data:") {
		if url, saveErr := saveBase64Media(img3); saveErr == nil {
			img3 = url
		}
	}
	if strings.HasPrefix(img4, "data:") {
		if url, saveErr := saveBase64Media(img4); saveErr == nil {
			img4 = url
		}
	}

	if img1 == "" {
		img1 = "https://via.placeholder.com/300"
	}
	if category == "" {
		category = "Bidhaa"
	}
	if location == "" {
		location = "Tanzania"
	}

	_, err := db.Exec("INSERT INTO designs (title, description, price, image_url, image_url2, image_url3, image_url4, video_url, category, designer_name, location, vendor_phone, status, rejection_reason) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'approved', '')",
		title, description, price, img1, img2, img3, img4, videoURL, category, designerName, location, vendorPhone)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuhifadhi bidhaa"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imechapishwa kwenye Soko Kuu kwa mafanikio!"})
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
		Category    string  `json:"category"`
	}

	r.Body = http.MaxBytesReader(w, r.Body, 80<<20)
	json.NewDecoder(r.Body).Decode(&payload)

	w.Header().Set("Content-Type", "application/json")
	_, err := db.Exec("UPDATE designs SET title = $1, description = $2, price = $3, image_url = $4, image_url2 = $5, image_url3 = $6, image_url4 = $7, category = $8 WHERE id = $9",
		payload.Title, payload.Description, payload.Price, payload.ImageURL, payload.ImageURL2, payload.ImageURL3, payload.ImageURL4, payload.Category, payload.ID)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kusasisha"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imesasishwa kikamilifu!"})
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
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imefutwa!"})
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
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Oda imepokelewa!"})
}

// ----------------- API ZA HADITHI (REKODI NA KUHUSISHA STORYTELLER) -----------------

func getStoriesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, content, COALESCE(cover_image, ''), storyteller_name, status, rejection_reason, created_at FROM stories WHERE status = 'approved' ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana kusoma hadithi", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.CoverImage, &s.StorytellerName, &s.Status, &s.RejectionReason, &s.CreatedAt); err != nil {
			continue
		}
		stories = append(stories, s)
	}

	if stories == nil {
		stories = []Story{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stories)
}

func getMyStoriesHandler(w http.ResponseWriter, r *http.Request) {
	storyteller := r.URL.Query().Get("storyteller")
	if storyteller == "" {
		storyteller = r.URL.Query().Get("designer")
	}

	rows, err := db.Query("SELECT id, title, content, COALESCE(cover_image, ''), storyteller_name, status, rejection_reason, created_at FROM stories WHERE storyteller_name = $1 ORDER BY id DESC", storyteller)
	if err != nil {
		http.Error(w, "Imeshindikana kusoma hadithi zako", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.CoverImage, &s.StorytellerName, &s.Status, &s.RejectionReason, &s.CreatedAt); err != nil {
			continue
		}
		stories = append(stories, s)
	}

	if stories == nil {
		stories = []Story{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stories)
}

func uploadStoryJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var title, content, coverImage, storytellerName string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Title           string `json:"title"`
			Content         string `json:"content"`
			CoverImage      string `json:"cover_image"`
			StorytellerName string `json:"storyteller_name"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			title = payload.Title
			content = payload.Content
			coverImage = payload.CoverImage
			storytellerName = payload.StorytellerName
		}
	}

	if title == "" {
		r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
		r.ParseMultipartForm(50 << 20)
		r.ParseForm()
		title = r.FormValue("title")
		content = r.FormValue("content")
		coverImage = r.FormValue("cover_image")
		if coverImage == "" {
			coverImage = r.FormValue("image_url")
		}
		storytellerName = r.FormValue("storyteller_name")
		if storytellerName == "" {
			storytellerName = r.FormValue("designer_name")
		}
	}

	if storytellerName != "" {
		if ok, errMsg := checkSubscriptionAndVerification(storytellerName); !ok {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": errMsg,
			})
			return
		}
	}

	if strings.HasPrefix(coverImage, "data:") {
		if url, saveErr := saveBase64Media(coverImage); saveErr == nil {
			coverImage = url
		}
	}

	_, err := db.Exec("INSERT INTO stories (title, content, cover_image, storyteller_name, status, rejection_reason) VALUES ($1, $2, $3, $4, 'approved', '')",
		title, content, coverImage, storytellerName)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kuhifadhi hadithi: " + err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Hadithi yako imechapishwa kikamilifu kwenye Ukurasa wa Hadithi!"})
}

func deleteMyStoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	storyID := r.URL.Query().Get("id")
	_, err := db.Exec("DELETE FROM stories WHERE id = $1", storyID)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta hadithi"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Hadithi imefutwa!"})
}

// ----------------- ADMIN API -----------------

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

func createSubAdminHandler(w http.ResponseWriter, r *http.Request) {
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

	if username == "" || password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina na nenosiri la sub-admin!"})
		return
	}

	_, err := db.Exec(`
		INSERT INTO app_accounts (username, password, role, verification_status, subscription_status)
		VALUES ($1, $2, 'sub_admin', 'approved', 'active')`,
		username, password)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina hili linatumika tayari au kosa la database!"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Sub-Admin ameongezwa kikamilifu kwenye mfumo!",
	})
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
	_, err := db.Exec("UPDATE designs SET status = 'approved', rejection_reason = '' WHERE id = $1", id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imeidhinishwa!"})
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
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imekataliwa!"})
}

func adminDeleteDesignHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	designID := r.URL.Query().Get("id")
	if designID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "ID ya bidhaa inahitajika!"})
		return
	}

	_, err := db.Exec("DELETE FROM designs WHERE id = $1", designID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta bidhaa kwenye database"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, 
		"message": "Bidhaa imefutwa kikamilifu na Admin!",
	})
}

func adminGetStoriesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, content, COALESCE(cover_image, ''), storyteller_name, status, rejection_reason, created_at FROM stories ORDER BY id DESC")
	if err != nil {
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.CoverImage, &s.StorytellerName, &s.Status, &s.RejectionReason, &s.CreatedAt); err != nil {
			continue
		}
		stories = append(stories, s)
	}

	if stories == nil {
		stories = []Story{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stories)
}

func adminApproveStoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	id := r.URL.Query().Get("id")
	_, err := db.Exec("UPDATE stories SET status = 'approved', rejection_reason = '' WHERE id = $1", id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Hadithi imeidhinishwa!"})
}

func adminRejectStoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	w.Header().Set("Content-Type", "application/json")
	id := r.URL.Query().Get("id")
	reason := r.FormValue("reason")
	if reason == "" {
		reason = "Haikutimiza vigezo vya hadithi."
	}

	_, err := db.Exec("UPDATE stories SET status = 'rejected', rejection_reason = $1 WHERE id = $2", reason, id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Hadithi imekataliwa!"})
}

func adminDeleteStoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false, 
		"message": "Sub-admin hana ruhusa ya kufuta hadithi!",
	})
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
	rows, err := db.Query("SELECT id, username, role, verification_status, COALESCE(id_type, ''), COALESCE(id_number, ''), COALESCE(id_image_url, ''), COALESCE(rejection_reason, ''), COALESCE(subscription_status, 'free'), trial_ends_at, COALESCE(payment_phone, ''), COALESCE(payment_name, ''), created_at FROM app_accounts ORDER BY id DESC")
	if err != nil {
		w.Write([]byte(`[]`))
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var trialEnds sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.IDType, &u.IDNumber, &u.IDImageURL, &u.RejectionReason, &u.SubscriptionStatus, &trialEnds, &u.PaymentPhone, &u.PaymentName, &u.CreatedAt); err != nil {
			continue
		}
		if trialEnds.Valid {
			u.TrialEndsAt = trialEnds.Time.Format("2006-01-02 15:04:05")
		}
		users = append(users, u)
	}

	if users == nil {
		users = []User{}
	}

	json.NewEncoder(w).Encode(users)
}

func adminGetBuyersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.Query("SELECT id, username, role, verification_status, created_at FROM app_accounts WHERE role = 'buyer' ORDER BY id DESC")
	if err != nil {
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

func adminGetSellersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.Query("SELECT id, username, role, verification_status, COALESCE(id_type, ''), COALESCE(id_number, ''), COALESCE(id_image_url, ''), COALESCE(rejection_reason, ''), COALESCE(subscription_status, 'free'), trial_ends_at, COALESCE(payment_phone, ''), COALESCE(payment_name, ''), created_at FROM app_accounts WHERE role = 'seller' ORDER BY id DESC")
	if err != nil {
		w.Write([]byte(`[]`))
		return
	}
	defer rows.Close()

	var sellers []User
	for rows.Next() {
		var u User
		var trialEnds sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.IDType, &u.IDNumber, &u.IDImageURL, &u.RejectionReason, &u.SubscriptionStatus, &trialEnds, &u.PaymentPhone, &u.PaymentName, &u.CreatedAt); err != nil {
			continue
		}
		if trialEnds.Valid {
			u.TrialEndsAt = trialEnds.Time.Format("2006-01-02 15:04:05")
		}
		sellers = append(sellers, u)
	}

	if sellers == nil {
		sellers = []User{}
	}

	json.NewEncoder(w).Encode(sellers)
}

func adminGetStorytellersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.Query("SELECT id, username, role, verification_status, COALESCE(id_type, ''), COALESCE(id_number, ''), COALESCE(id_image_url, ''), COALESCE(rejection_reason, ''), COALESCE(subscription_status, 'free'), trial_ends_at, COALESCE(payment_phone, ''), COALESCE(payment_name, ''), created_at FROM app_accounts WHERE role = 'storyteller' ORDER BY id DESC")
	if err != nil {
		w.Write([]byte(`[]`))
		return
	}
	defer rows.Close()

	var storytellers []User
	for rows.Next() {
		var u User
		var trialEnds sql.NullTime
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.VerificationStatus, &u.IDType, &u.IDNumber, &u.IDImageURL, &u.RejectionReason, &u.SubscriptionStatus, &trialEnds, &u.PaymentPhone, &u.PaymentName, &u.CreatedAt); err != nil {
			continue
		}
		if trialEnds.Valid {
			u.TrialEndsAt = trialEnds.Time.Format("2006-01-02 15:04:05")
		}
		storytellers = append(storytellers, u)
	}

	if storytellers == nil {
		storytellers = []User{}
	}

	json.NewEncoder(w).Encode(storytellers)
}

func adminApproveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	userID := r.URL.Query().Get("id")
	_, err := db.Exec("UPDATE app_accounts SET verification_status = 'approved', rejection_reason = '' WHERE id = $1", userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imepitishwa!"})
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
		reason = "Imekataliwa."
	}

	_, err := db.Exec("UPDATE app_accounts SET verification_status = 'rejected', rejection_reason = $1 WHERE id = $2", reason, userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana"})
		return
	}

    json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Imekataliwa!"})
}

func adminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	userID := r.URL.Query().Get("id")
	if userID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "ID ya mtumiaji inahitajika!"})
		return
	}

	_, err := db.Exec("DELETE FROM app_accounts WHERE id = $1", userID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta mtumiaji kwenye database"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true, 
		"message": "Mtumiaji amefutwa kikamilifu na Admin!",
	})
}

func adminGetFeedbackHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rows, err := db.Query("SELECT id, name, COALESCE(email, ''), message, created_at FROM feedback ORDER BY id DESC")
	if err != nil {
		w.Write([]byte(`[]`))
		return
	}
	defer rows.Close()

	var feedbacks []Feedback
	for rows.Next() {
		var f Feedback
		if err := rows.Scan(&f.ID, &f.Name, &f.Email, &f.Message, &f.CreatedAt); err != nil {
			continue
		}
		feedbacks = append(feedbacks, f)
	}

	if feedbacks == nil {
		feedbacks = []Feedback{}
	}

	json.NewEncoder(w).Encode(feedbacks)
}
