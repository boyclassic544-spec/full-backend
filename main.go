package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var jwtSecret []byte

// ================== VALIDATION REGEX ==================
var (
	phoneRegex = regexp.MustCompile(`^0[67]\d{8}$`)
	nidaRegex  = regexp.MustCompile(`^\d{20}$`)
)

// ================== STRUCTS ==================
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
	ID              int     `json:"id"`
	Title           string  `json:"title"`
	Content         string  `json:"content"`
	CoverImage      string  `json:"cover_image"`
	StorytellerName string  `json:"storyteller_name"`
	Status          string  `json:"status"`
	RejectionReason string  `json:"rejection_reason"`
	CreatedAt       string  `json:"created_at"`
	Price           float64 `json:"price"`
	IsPaid          bool    `json:"is_paid"`
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

// ================== MAIN ==================
func main() {
	var err error

	// JWT Secret
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "sokosmart-tz-super-secret-key-2026-khalid-secure-fixed"
		log.Println("⚠️  JWT_SECRET haijawekwa. Inatumia default.")
	}
	jwtSecret = []byte(secret)

	// Database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:password@localhost:5432/sokosmart?sslmode=disable"
		log.Println("⚠️  DATABASE_URL haijawekwa, inatumia default ya localhost.")
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
	seedSuperAdmin()

	// ============ PUBLIC ROUTES ============
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/signup", signupHandler)
	http.HandleFunc("/api/signin", signinHandler)
	http.HandleFunc("/api/profile", profileHandler)
	http.HandleFunc("/api/submit-subscription-payment", submitSubscriptionPaymentHandler)

	// ============ SELLER ROUTES ============
	http.HandleFunc("/api/designs", getDesignsHandler)
	http.HandleFunc("/api/my-designs", getMyDesignsHandler)
	http.HandleFunc("/api/upload", uploadDesignJSONHandler)
	http.HandleFunc("/api/update-design", updateDesignJSONHandler)
	http.HandleFunc("/api/delete-design", deleteMyDesignHandler)
	http.HandleFunc("/api/buy", buyDesignHandler)

	// ============ STORY ROUTES ============
	http.HandleFunc("/api/stories", getStoriesHandler)
	http.HandleFunc("/api/my-stories", getMyStoriesHandler)
	http.HandleFunc("/api/storyteller/upload", uploadStoryJSONHandler)
	http.HandleFunc("/api/delete-story", deleteMyStoryHandler)

	// ============ ADMIN ROUTES ============
	http.HandleFunc("/api/admin/login", adminLoginHandler)

	// Design management
	http.HandleFunc("/api/admin/designs", adminAuthMiddleware(adminGetDesignsHandler))
	http.HandleFunc("/api/admin/approve", adminAuthMiddleware(adminApproveDesignHandler))
	http.HandleFunc("/api/admin/reject", adminAuthMiddleware(adminRejectDesignHandler))
	http.HandleFunc("/api/admin/delete-design", adminAuthMiddleware(adminDeleteDesignHandler))

	// Story management
	http.HandleFunc("/api/admin/stories", adminAuthMiddleware(adminGetStoriesHandler))
	http.HandleFunc("/api/admin/approve-story", adminAuthMiddleware(adminApproveStoryHandler))
	http.HandleFunc("/api/admin/reject-story", adminAuthMiddleware(adminRejectStoryHandler))
	http.HandleFunc("/api/admin/delete-story", adminAuthMiddleware(adminDeleteStoryHandler))

	// Orders & Users
	http.HandleFunc("/api/admin/orders", adminAuthMiddleware(adminGetOrdersHandler))
	http.HandleFunc("/api/admin/users", adminAuthMiddleware(adminUsersHandler))
	http.HandleFunc("/api/admin/buyers", adminAuthMiddleware(adminGetBuyersHandler))
	http.HandleFunc("/api/admin/sellers", adminAuthMiddleware(adminGetSellersHandler))
	http.HandleFunc("/api/admin/storytellers", adminAuthMiddleware(adminGetStorytellersHandler))
	http.HandleFunc("/api/admin/approve-user", adminAuthMiddleware(adminApproveUserHandler))
	http.HandleFunc("/api/admin/reject-user", adminAuthMiddleware(adminRejectUserHandler))
	http.HandleFunc("/api/admin/delete-user", adminAuthMiddleware(adminDeleteUserHandler))
	http.HandleFunc("/api/admin/approve-subscription", adminAuthMiddleware(adminApproveSubscriptionHandler))

	// Admin management
	http.HandleFunc("/api/admin/add-admin", adminAuthMiddleware(superAdminOnly(adminAddAdminHandler)))
	http.HandleFunc("/api/admin/list-admins", adminAuthMiddleware(superAdminOnly(adminListAdminsHandler)))
	http.HandleFunc("/api/admin/delete-admin", adminAuthMiddleware(superAdminOnly(adminDeleteAdminHandler)))

	// ============ SERVER START ============
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	handler := corsMiddleware(loggingMiddleware(http.DefaultServeMux))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	fmt.Printf("✅ Seva inaanza kusikiliza kwenye bandari %s...\n", port)
	log.Fatal(srv.ListenAndServe())
}

// ================== DB INIT ==================
func initDB() {
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
	if _, err := db.Exec(queryUsers); err != nil {
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
	if _, err := db.Exec(queryDesigns); err != nil {
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		price NUMERIC DEFAULT 0,
		is_paid BOOLEAN DEFAULT FALSE
	);`
	if _, err := db.Exec(queryStories); err != nil {
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
	if _, err := db.Exec(queryOrders); err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la orders: %v", err)
	}

	queryAdmins := `
	CREATE TABLE IF NOT EXISTS admins (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		admin_level TEXT DEFAULT 'sub',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(queryAdmins); err != nil {
		log.Fatalf("Imeshindikana kutengeneza jedwali la admins: %v", err)
	}

	// Safe column additions
	db.Exec("ALTER TABLE stories ADD COLUMN IF NOT EXISTS cover_image TEXT DEFAULT '';")
	db.Exec("ALTER TABLE stories ADD COLUMN IF NOT EXISTS price NUMERIC DEFAULT 0;")
	db.Exec("ALTER TABLE stories ADD COLUMN IF NOT EXISTS is_paid BOOLEAN DEFAULT FALSE;")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS subscription_status TEXT DEFAULT 'free';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMP;")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS payment_phone TEXT DEFAULT '';")
	db.Exec("ALTER TABLE app_accounts ADD COLUMN IF NOT EXISTS payment_name TEXT DEFAULT '';")
}

// ================== SEED SUPER ADMIN ==================
func seedSuperAdmin() {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")

	if username == "" {
		username = "khalid"
	}
	if password == "" {
		password = "khalid_secret_2026@"
	}

	log.Println("═══════════════════════════════════════════════")
	log.Println("🔧 SEEDING SUPER ADMIN...")
	log.Printf("   👤 Username: %s", username)
	log.Println("   🔑 Password: [imefichwa kwa usalama]")
	log.Println("═══════════════════════════════════════════════")

	var existingID int
	var existingHash string
	err := db.QueryRow("SELECT id, password FROM admins WHERE username = $1", username).Scan(&existingID, &existingHash)

	if err == nil {
		if bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(password)) == nil {
			log.Printf("ℹ️  Admin '%s' tayari yupo na password ni sahihi.", username)
			log.Println("═══════════════════════════════════════════════")
			return
		}
		newHash, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if hashErr != nil {
			log.Printf("❌ Imeshindikana ku-hash password mpya: %v", hashErr)
			return
		}
		_, updateErr := db.Exec("UPDATE admins SET password = $1, admin_level = 'super' WHERE username = $2", string(newHash), username)
		if updateErr != nil {
			log.Printf("❌ Imeshindikana kusasisha password: %v", updateErr)
			return
		}
		log.Printf("✅ Password ya admin '%s' imesasishwa!", username)
		log.Println("═══════════════════════════════════════════════")
		return
	}

	hashed, hashErr := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if hashErr != nil {
		log.Printf("❌ Imeshindikana ku-hash nenosiri la admin: %v", hashErr)
		return
	}

	_, insertErr := db.Exec("INSERT INTO admins (username, password, admin_level) VALUES ($1, $2, 'super')",
		username, string(hashed))
	if insertErr != nil {
		log.Printf("❌ Imeshindikana kuunda Super Admin: %v", insertErr)
		return
	}

	log.Println("═══════════════════════════════════════════════")
	log.Println("✅ SUPER ADMIN AMEUNDWA KWA MAFANIKIO!")
	log.Println("═══════════════════════════════════════════════")
	log.Printf("   👤 Username: %s", username)
	log.Println("   🔑 Password: [imefichwa kwa usalama]")
	log.Println("═══════════════════════════════════════════════")
}

// ================== MIDDLEWARE ==================

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func adminAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, "message": "Hakuna ruhusa. Tafadhali ingia kama Admin.",
			})
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("njia ya kusaini si sahihi")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, "message": "Token si sahihi au imeisha muda wake.",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false, "message": "Token haina taarifa sahihi.",
			})
			return
		}

		level, _ := claims["level"].(string)
		username, _ := claims["username"].(string)
		r.Header.Set("X-Admin-Level", level)
		r.Header.Set("X-Admin-Username", username)

		next(w, r)
	}
}

func superAdminOnly(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		level := r.Header.Get("X-Admin-Level")
		if level != "super" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Hii operesheni inaruhusiwa kwa Super Admin pekee!",
			})
			return
		}
		next(w, r)
	}
}

// ================== UTILITIES ==================

// ✅ Inatuma picha Cloudinary kwa kutumia CLOUDINARY_URL
func saveBase64Media(dataURL string) (string, error) {
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")
	if cloudinaryURL == "" {
		return "", fmt.Errorf("CLOUDINARY_URL haijawekwa kwenye environment variables")
	}

	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		return "", fmt.Errorf("cloudinary config error: %v", err)
	}

	ctx := context.Background()
	resp, err := cld.Upload.Upload(ctx, dataURL, uploader.UploadParams{
		Folder: "sokosmart",
	})
	if err != nil {
		return "", fmt.Errorf("cloudinary upload error: %v", err)
	}

	return resp.SecureURL, nil
}

func sanitize(s string) string {
	return strings.TrimSpace(html.EscapeString(s))
}

func validPhone(phone string) bool {
	return phoneRegex.MatchString(strings.TrimSpace(phone))
}

func validNIDA(nida string) bool {
	return nida == "" || nidaRegex.MatchString(strings.TrimSpace(nida))
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/index.html")
}

// ================== AUTH ==================

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
		r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
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
		r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
		r.ParseMultipartForm(20 << 20)
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

	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina la mtumiaji na nenosiri!"})
		return
	}

	if len(password) < 6 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Nenosiri liwe na angalau herufi 6!"})
		return
	}

	if role == "" {
		role = "buyer"
	}
	if role == "stela" {
		role = "storyteller"
	}
	if role != "buyer" && role != "seller" && role != "storyteller" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Aina ya akaunti si sahihi!"})
		return
	}

	if role == "seller" || role == "storyteller" {
		if idNumber != "" && !validNIDA(idNumber) {
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Namba ya NIDA iwe na tarakimu 20!"})
			return
		}
	}

	var verificationStatus = "approved"
	if role == "seller" || role == "storyteller" {
		verificationStatus = "pending"
	}

	var idImageURL = ""
	if rawIDImage != "" && strings.HasPrefix(rawIDImage, "data:") {
		if savedURL, saveErr := saveBase64Media(rawIDImage); saveErr == nil {
			idImageURL = savedURL
		} else {
			log.Printf("⚠️  Kosa la kuhifadhi picha ya kitambulisho: %v", saveErr)
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("❌ Kosa la bcrypt: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Hitilafu ya mfumo. Jaribu tena."})
		return
	}

	trialEnds := time.Now().Add(30 * 24 * time.Hour)
	subStatus := "free"
	if role == "buyer" {
		subStatus = "active"
	}

	_, err = db.Exec(`
		INSERT INTO app_accounts (username, password, role, verification_status, id_type, id_number, id_image_url, subscription_status, trial_ends_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		sanitize(username), string(hashedPassword), role, verificationStatus, sanitize(idType), sanitize(idNumber), idImageURL, subStatus, trialEnds)

	if err != nil {
		log.Printf("❌ Kosa la kusajili: %v", err)
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

	username = strings.TrimSpace(username)

	var storedPass, role, verificationStatus, rejectionReason, subStatus string
	var trialEnds sql.NullTime
	err := db.QueryRow("SELECT password, role, verification_status, COALESCE(rejection_reason, ''), COALESCE(subscription_status, 'free'), trial_ends_at FROM app_accounts WHERE username = $1", username).
		Scan(&storedPass, &role, &verificationStatus, &rejectionReason, &subStatus, &trialEnds)

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la mtumiaji au nenosiri si sahihi!"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedPass), []byte(password)); err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la mtumiaji au nenosiri si sahihi!"})
		return
	}

	isExpired := false
	if role != "buyer" && trialEnds.Valid && time.Now().After(trialEnds.Time) && subStatus != "active" {
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
	} else {
		u.TrialEndsAt = "Haipo"
	}

	isExpired := false
	if u.Role != "buyer" && trialEnds.Valid && time.Now().After(trialEnds.Time) && u.SubscriptionStatus != "active" {
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

	if !validPhone(paymentPhone) {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Namba ya simu si sahihi! Tumia muundo 07XXXXXXXX."})
		return
	}

	_, err := db.Exec("UPDATE app_accounts SET payment_phone = $1, payment_name = $2, subscription_status = 'pending_payment' WHERE username = $3",
		sanitize(paymentPhone), sanitize(paymentName), sanitize(username))
	if err != nil {
		log.Printf("❌ Kosa la submit payment: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kutuma taarifa za malipo"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Ombi lako la malipo limetumwa kwa Admin! Tafadhali subiri uhakiki.",
	})
}

// ================== SUBSCRIPTION CHECK ==================

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

	if role == "buyer" {
		return true, ""
	}

	if trialEnds.Valid && time.Now().After(trialEnds.Time) && subStatus != "active" {
		return false, "Mwezi wako wa bure umekwisha. Tafadhali lipa Tsh 5,000 kwenda Tigo Lipa: 45416553 (SALMIN TAMIMU HUSEIN) kisha utume namba yako na jina ili Admin akujulishe."
	}

	return true, ""
}

// ================== DESIGNS ==================

func getDesignsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, price, image_url, COALESCE(image_url2,''), COALESCE(image_url3,''), COALESCE(image_url4,''), video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs WHERE status = 'approved' ORDER BY id DESC")
	if err != nil {
		log.Printf("❌ Kosa la getDesigns: %v", err)
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
		log.Printf("❌ Kosa la getMyDesigns: %v", err)
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
		fmt.Sscanf(r.FormValue("price"), "%f", &price)
		img1 = r.FormValue("image_url")
		if img1 == "" {
			img1 = r.FormValue("image")
		}
		img2 = r.FormValue("image_url2")
		img3 = r.FormValue("image_url3")
		img4 = r.FormValue("image_url4")
	}

	if strings.TrimSpace(title) == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la bidhaa linahitajika!"})
		return
	}
	if price < 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Bei haiwezi kuwa hasi!"})
		return
	}
	if vendorPhone != "" && !validPhone(vendorPhone) {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Namba ya simu si sahihi! Tumia muundo 07XXXXXXXX."})
		return
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

	for _, img := range []*string{&img1, &img2, &img3, &img4} {
		if strings.HasPrefix(*img, "data:") {
			if url, saveErr := saveBase64Media(*img); saveErr == nil {
				*img = url
			} else {
				log.Printf("⚠️  Kosa la kuhifadhi picha: %v", saveErr)
			}
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
		sanitize(title), sanitize(description), price, img1, img2, img3, img4, videoURL, sanitize(category), sanitize(designerName), sanitize(location), sanitize(vendorPhone))

	if err != nil {
		log.Printf("❌ Kosa la upload design: %v", err)
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

	if payload.Price < 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Bei haiwezi kuwa hasi!"})
		return
	}

	_, err := db.Exec("UPDATE designs SET title = $1, description = $2, price = $3, image_url = $4, image_url2 = $5, image_url3 = $6, image_url4 = $7, category = $8 WHERE id = $9",
		sanitize(payload.Title), sanitize(payload.Description), payload.Price, payload.ImageURL, payload.ImageURL2, payload.ImageURL3, payload.ImageURL4, sanitize(payload.Category), payload.ID)

	if err != nil {
		log.Printf("❌ Kosa la update design: %v", err)
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
		log.Printf("❌ Kosa la delete design: %v", err)
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

	if !validPhone(phone) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Namba ya simu si sahihi!"})
		return
	}

	_, err := db.Exec("INSERT INTO orders (design_id, phone, amount, payment_status) VALUES ($1, $2, $3, 'pending_tigo_lipa')", designID, phone, amount)
	if err != nil {
		log.Printf("❌ Kosa la buy: %v", err)
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Oda imepokelewa!"})
}

// ================== STORIES ==================

func getStoriesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, content, COALESCE(cover_image, ''), storyteller_name, status, COALESCE(rejection_reason, ''), created_at, price, is_paid FROM stories WHERE status = 'approved' ORDER BY id DESC")
	if err != nil {
		log.Printf("❌ Kosa la getStories: %v", err)
		http.Error(w, "Imeshindikana kusoma hadithi", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.CoverImage, &s.StorytellerName, &s.Status, &s.RejectionReason, &s.CreatedAt, &s.Price, &s.IsPaid); err != nil {
			log.Printf("Kosa la kuscan hadithi: %v", err)
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

	rows, err := db.Query("SELECT id, title, content, COALESCE(cover_image, ''), storyteller_name, status, COALESCE(rejection_reason, ''), created_at, price, is_paid FROM stories WHERE storyteller_name = $1 ORDER BY id DESC", storyteller)
	if err != nil {
		log.Printf("❌ Kosa la getMyStories: %v", err)
		http.Error(w, "Imeshindikana kusoma hadithi zako", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.CoverImage, &s.StorytellerName, &s.Status, &s.RejectionReason, &s.CreatedAt, &s.Price, &s.IsPaid); err != nil {
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
	var price float64
	var isPaid bool

	r.Body = http.MaxBytesReader(w, r.Body, 60<<20)

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Title           string  `json:"title"`
			Content         string  `json:"content"`
			CoverImage      string  `json:"cover_image"`
			StorytellerName string  `json:"storyteller_name"`
			Price           float64 `json:"price"`
			IsPaid          bool    `json:"is_paid"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			log.Printf("❌ Kosa la kusoma JSON: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"message": "Taarifa zilizotumwa si sahihi. Jaribu tena.",
			})
			return
		}

		title = payload.Title
		content = payload.Content
		coverImage = payload.CoverImage
		storytellerName = payload.StorytellerName
		price = payload.Price
		isPaid = payload.IsPaid

		log.Printf("📥 [STORY UPLOAD] title=%q, is_paid=%v, price=%.2f, cover_size=%d chars, storyteller=%q",
			title, isPaid, price, len(coverImage), storytellerName)
	}

	if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Kichwa na maudhui ya hadithi vinahitajika!"})
		return
	}
	if price < 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Bei haiwezi kuwa hasi!"})
		return
	}

	if isPaid && price <= 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Kwa hadithi ya kulipia, bei lazima iwe zaidi ya 0!"})
		return
	}

	if storytellerName == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la mwandishi linahitajika!"})
		return
	}

	if ok, errMsg := checkSubscriptionAndVerification(storytellerName); !ok {
		log.Printf("❌ [STORY UPLOAD] Verification failed kwa %q: %s", storytellerName, errMsg)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": errMsg,
		})
		return
	}

	if coverImage != "" {
		if strings.HasPrefix(coverImage, "data:") {
			savedURL, saveErr := saveBase64Media(coverImage)
			if saveErr != nil {
				log.Printf("❌ [STORY UPLOAD] Kosa la kuhifadhi cover: %v", saveErr)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "Imeshindikana kuhifadhi picha ya cover: " + saveErr.Error(),
				})
				return
			}
			coverImage = savedURL
			log.Printf("✅ [STORY UPLOAD] Cover imehifadhiwa: %s", coverImage)
		}
	} else {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Picha ya cover inahitajika!",
		})
		return
	}

	_, err := db.Exec("INSERT INTO stories (title, content, cover_image, storyteller_name, status, price, is_paid) VALUES ($1, $2, $3, $4, 'approved', $5, $6)",
		sanitize(title), content, coverImage, sanitize(storytellerName), price, isPaid)

	if err != nil {
		log.Printf("❌ [STORY UPLOAD] Kosa la kuhifadhi hadithi: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"message": "Imeshindikana kuhifadhi hadithi: " + err.Error(),
		})
		return
	}

	log.Printf("✅ [STORY UPLOAD] Hadithi ya %q imechapishwa kwa mafanikio!", title)

	msg := "Hadithi yako imechapishwa kikamilifu!"
	if isPaid {
		msg = fmt.Sprintf("Hadithi yako ya kulipia (TZS %.0f) imechapishwa kikamilifu!", price)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": msg,
	})
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
		log.Printf("❌ Kosa la delete story: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta hadithi"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Hadithi imefutwa!"})
}

// ================== ADMIN LOGIN (JWT) ==================

func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
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

	username = strings.TrimSpace(username)

	log.Printf("🔍 [ADMIN LOGIN] Attempt: username=%q, password_length=%d", username, len(password))

	if username == "" || password == "" {
		log.Println("❌ [ADMIN LOGIN] Username au password tupu")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina na nenosiri!"})
		return
	}

	var storedPass, adminLevel string
	err := db.QueryRow("SELECT password, admin_level FROM admins WHERE username = $1", username).
		Scan(&storedPass, &adminLevel)

	if err != nil {
		log.Printf("❌ [ADMIN LOGIN] Admin '%s' hajapatikana: %v", username, err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina au nenosiri si sahihi!"})
		return
	}

	log.Printf("✅ [ADMIN LOGIN] Admin '%s' amepatikana, akikagua password...", username)

	if err := bcrypt.CompareHashAndPassword([]byte(storedPass), []byte(password)); err != nil {
		log.Printf("❌ [ADMIN LOGIN] Password si sahihi kwa admin '%s': %v", username, err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina au nenosiri si sahihi!"})
		return
	}

	log.Printf("✅ [ADMIN LOGIN] Password ni sahihi! Kutengeneza token...")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"level":    adminLevel,
		"exp":      time.Now().Add(8 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("❌ [ADMIN LOGIN] Kosa la kutengeneza token: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Hitilafu ya mfumo"})
		return
	}

	log.Printf("✅ [ADMIN LOGIN] Token imetengenezwa kwa admin '%s'!", username)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "Umeingia kama Admin!",
		"token":       tokenStr,
		"username":    username,
		"admin_level": adminLevel,
	})
}

// ================== ADMIN MANAGEMENT ==================

func adminAddAdminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var username, password, level string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var payload struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Level    string `json:"admin_level"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			username = payload.Username
			password = payload.Password
			level = payload.Level
		}
	}

	if username == "" {
		r.ParseForm()
		username = r.FormValue("username")
		password = r.FormValue("password")
		level = r.FormValue("admin_level")
	}

	username = strings.TrimSpace(username)

	if username == "" || password == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jaza jina na nenosiri!"})
		return
	}

	if len(password) < 6 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Nenosiri liwe na angalau herufi 6!"})
		return
	}

	if level != "super" && level != "sub" {
		level = "sub"
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Hitilafu ya mfumo"})
		return
	}

	_, err = db.Exec("INSERT INTO admins (username, password, admin_level) VALUES ($1, $2, $3)",
		sanitize(username), string(hashed), level)
	if err != nil {
		log.Printf("❌ Kosa la kuongeza admin: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Jina la admin linatumika tayari!"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Admin ameongezwa vizuri kabisa!"})
}

func adminListAdminsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := db.Query("SELECT id, username, admin_level FROM admins ORDER BY id DESC")
	if err != nil {
		log.Printf("❌ Kosa la list admins: %v", err)
		w.Write([]byte(`[]`))
		return
	}
	defer rows.Close()

	var admins []map[string]interface{}
	for rows.Next() {
		var id int
		var username, role string
		if err := rows.Scan(&id, &username, &role); err != nil {
			continue
		}
		admins = append(admins, map[string]interface{}{
			"id":          id,
			"username":    username,
			"admin_level": role,
		})
	}

	if admins == nil {
		admins = []map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(admins)
}

func adminDeleteAdminHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "ID inahitajika!"})
		return
	}

	var superCount int
	db.QueryRow("SELECT COUNT(*) FROM admins WHERE admin_level = 'super'").Scan(&superCount)

	var targetLevel string
	err := db.QueryRow("SELECT admin_level FROM admins WHERE id = $1", id).Scan(&targetLevel)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Admin hajapatikana!"})
		return
	}

	if targetLevel == "super" && superCount <= 1 {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Hauwezi kufuta Super Admin wa mwisho!"})
		return
	}

	_, err = db.Exec("DELETE FROM admins WHERE id = $1", id)
	if err != nil {
		log.Printf("❌ Kosa la delete admin: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta admin"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Admin amefutwa!"})
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
		log.Printf("❌ Kosa la approve sub: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kusasisha usajili"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Hongera! Malipo yamethibitishwa na akaunti imeongezewa mwezi 1 mpya kikamilifu.",
	})
}

// ================== ADMIN DESIGN HANDLERS ==================

func adminGetDesignsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, description, price, image_url, COALESCE(image_url2,''), COALESCE(image_url3,''), COALESCE(image_url4,''), video_url, category, designer_name, location, vendor_phone, status, rejection_reason FROM designs ORDER BY id DESC")
	if err != nil {
		log.Printf("❌ Kosa la adminGetDesigns: %v", err)
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

	_, err := db.Exec("UPDATE designs SET status = 'rejected', rejection_reason = $1 WHERE id = $2", sanitize(reason), id)
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
		log.Printf("❌ Kosa la admin delete design: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta bidhaa"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Bidhaa imefutwa kikamilifu na Admin!"})
}

// ================== ADMIN STORY HANDLERS ==================

func adminGetStoriesHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, content, COALESCE(cover_image, ''), storyteller_name, status, COALESCE(rejection_reason, ''), created_at, price, is_paid FROM stories ORDER BY id DESC")
	if err != nil {
		log.Printf("❌ Kosa la adminGetStories: %v", err)
		http.Error(w, "Imeshindikana", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var stories []Story
	for rows.Next() {
		var s Story
		if err := rows.Scan(&s.ID, &s.Title, &s.Content, &s.CoverImage, &s.StorytellerName, &s.Status, &s.RejectionReason, &s.CreatedAt, &s.Price, &s.IsPaid); err != nil {
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

	_, err := db.Exec("UPDATE stories SET status = 'rejected', rejection_reason = $1 WHERE id = $2", sanitize(reason), id)
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
	storyID := r.URL.Query().Get("id")
	if storyID == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "ID ya hadithi inahitajika!"})
		return
	}

	_, err := db.Exec("DELETE FROM stories WHERE id = $1", storyID)
	if err != nil {
		log.Printf("❌ Kosa la admin delete story: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta hadithi"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Hadithi imefutwa kikamilifu na Admin Mkuu!"})
}

// ================== ADMIN ORDERS & USERS ==================

func adminGetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, design_id, phone, amount, payment_status, created_at FROM orders ORDER BY id DESC")
	if err != nil {
		log.Printf("❌ Kosa la adminGetOrders: %v", err)
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
		log.Printf("❌ Kosa la adminUsers: %v", err)
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
		} else {
			u.TrialEndsAt = "Haipo"
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
		log.Printf("❌ Kosa la adminGetBuyers: %v", err)
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
		} else {
			u.TrialEndsAt = "Haipo"
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
		} else {
			u.TrialEndsAt = "Haipo"
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

	_, err := db.Exec("UPDATE app_accounts SET verification_status = 'rejected', rejection_reason = $1 WHERE id = $2", sanitize(reason), userID)
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
		log.Printf("❌ Kosa la admin delete user: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "message": "Imeshindikana kufuta mtumiaji"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Mtumiaji amefutwa kikamilifu na Admin!"})
} 
