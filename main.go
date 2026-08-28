package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type User struct {
	FullName    string `json:"fullName"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Phone       string `json:"phone"`
}

type ServiceReq struct {
	FullName        string `json:"fullName"`
	Full_Name       string `json:"full_name"`
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	PhoneNumber     string `json:"phoneNumber"`
	Service         string `json:"service"`
	ServiceCategory string `json:"service_category"`
	Message         string `json:"message"`
	Description     string `json:"description"`
	CreatedAt       string `json:"createdAt"`
	Created_At      string `json:"created_at"`
}

type LoginReq struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// Muundo wa kumbukumbu za mashambulizi kwa ajili ya Admin Dashboard
type SecurityLog struct {
	IP        string `json:"ip"`
	Endpoint  string `json:"endpoint"`
	Payload   string `json:"payload"`
	Reason    string `json:"reason"`
	Timestamp string `json:"timestamp"`
}

var (
	mu           sync.Mutex
	users        = []User{}
	requests     = []ServiceReq{}
	securityLogs = []SecurityLog{} // Hapa ndipo tunatunza kumbukumbu za majaribio mabaya
)

// Kazi ya kuchunguza kama kuna dalili za mashambulizi (SQLi au XSS)
func detectAndLogAttack(r *http.Request, inputData string) bool {
	inputLower := strings.ToLower(inputData)
	dangerousPatterns := []string{
		"union select",
		"drop table",
		"<script>",
		"or 1=1",
		"exec(",
		"../",
	}

	detected := false
	reason := ""
	for _, pattern := range dangerousPatterns {
		if strings.Contains(inputLower, pattern) {
			detected = true
			reason = "Zana hatarishi imetundikwa: " + pattern
			break
		}
	}

	if detected {
		// Pata IP ya mshambulizi
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip, _, _ = net.SplitHostPort(r.RemoteAddr)
		}
		if ip == "" {
			ip	= "127.0.0.1"
		}
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = ip[:idx]
		}

		// Hifadhi kwenye kumbukumbu zetu za ndani
		mu.Lock()
		securityLogs = append(securityLogs, SecurityLog{
			IP:        ip,
			Endpoint:  r.URL.Path,
			Payload:   inputData,
			Reason:    reason,
			Timestamp: time.Now().Format("02 Jan 2006, 03:04:05 PM"),
		})
		mu.Unlock()
	}

	return detected
}

// Security Headers ya kupandisha grade iwe A
func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "admin.html")
	})

	mux.HandleFunc("/api/signup", handleSignup)
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/service-requests", handleService)
	mux.HandleFunc("/api/admin/users", handleGetUsers)
	mux.HandleFunc("/api/admin/requests", handleGetRequests)
	mux.HandleFunc("/api/admin/security-logs", handleGetSecurityLogs) // Route mpya ya kuonyesha mashambulizi kwenye Admin

	securedHandler := secureHeaders(mux)

	http.ListenAndServe(":"+port, securedHandler)
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	r.Body = http.MaxBytesReader(w, r.Body, 10240)

	var u User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Data si sahihi!"})
		return
	}

	// Kagua kama kuna shambulio kwenye maandishi yaliyoingizwa
	rawJSON, _ := json.Marshal(u)
	if detectAndLogAttack(r, string(rawJSON)) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Shambulio limegunduliwa na kuzuiwa!"})
		return
	}

	if u.FullName == "" {
		if u.Name != "" {
			u.FullName = u.Name
		} else {
			u.FullName = "Mteja Aliyejisajili"
		}
	}
	if u.Email == "" {
		u.Email = "haijulikani@domain.com"
	}

	mu.Lock()
	users = append(users, u)
	mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Imefaulu!"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	r.Body = http.MaxBytesReader(w, r.Body, 10240)

	var loginData LoginReq
	err := json.NewDecoder(r.Body).Decode(&loginData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Imeshindikana kusoma data ya kuingia!"})
		return
	}

	rawJSON, _ := json.Marshal(loginData)
	if detectAndLogAttack(r, string(rawJSON)) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Shambulio limegunduliwa na kuzuiwa!"})
		return
	}

	mu.Lock()
	found := false
	for _, u := range users {
		if (loginData.Email != "" && u.Email == loginData.Email) || (loginData.Phone != "" && (u.Phone == loginData.Phone || u.PhoneNumber == loginData.Phone)) {
			found = true
			break
		}
	}
	mu.Unlock()

	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Akaunti haipatikani! Tafadhali jisajili kwanza."})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Umefanikiwa kuingia!"})
}

func handleService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	r.Body = http.MaxBytesReader(w, r.Body, 20480)

	var req ServiceReq
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Ombi kubwa mno!"})
		return
	}

	rawJSON, _ := json.Marshal(req)
	if detectAndLogAttack(r, string(rawJSON)) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Shambulio limegunduliwa na kuzuiwa!"})
		return
	}

	if req.FullName == "" {
		if req.Full_Name != "" {
			req.FullName = req.Full_Name
		} else if req.Name != "" {
			req.FullName = req.Name
		} else {
			req.FullName = "Mteja"
		}
	}
	req.Full_Name = req.FullName

	if req.Phone == "" {
		if req.PhoneNumber != "" {
			req.Phone = req.PhoneNumber
		} else {
			req.Phone = "Namba haipo"
		}
	}
	req.PhoneNumber = req.Phone

	if req.Service == "" {
		if req.ServiceCategory != "" {
			req.Service = req.ServiceCategory
		} else {
			req.Service = "Huduma ya Jumla"
		}
	}
	req.ServiceCategory = req.Service

	if req.Message == "" {
		if req.Description != "" {
			req.Message = req.Description
		} else {
			req.Message = "Hakuna maelezo ya ziada"
		}
	}
	req.Description = req.Message

	nowStr := time.Now().Format("02 Jan 2006, 03:04 PM")
	req.CreatedAt = nowStr
	req.Created_At = nowStr

	mu.Lock()
	requests = append(requests, req)
	mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Ombi limepokelewa!"})
}

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	list := users
	mu.Unlock()
	json.NewEncoder(w).Encode(list)
}

func handleGetRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	list := requests
	mu.Unlock()
	json.NewEncoder(w).Encode(list)
}

// Hii inatuma orodha ya mashambulizi yaliyozuiwa kwenda kwenye Admin Dashboard
func handleGetSecurityLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	logs := securityLogs
	mu.Unlock()
	json.NewEncoder(w).Encode(logs)
}
// Update
