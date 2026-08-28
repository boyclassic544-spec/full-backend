package main

import (
	"encoding/json"
	"net/http"
	"os"
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

// Muundo mpya wa data zinazokuja wakati wa Login
type LoginReq struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

var (
	mu       sync.Mutex
	users    = []User{}
	requests = []ServiceReq{}
)

// secureHeaders Middleware ya kulinda tovuti na kuongeza Security Headers zote za A+
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
		http.ServeFile(w, r, "index.html")
	})

	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "admin.html")
	})

	mux.HandleFunc("/api/signup", handleSignup)
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/service-requests", handleService)
	mux.HandleFunc("/api/admin/users", handleGetUsers)
	mux.HandleFunc("/api/admin/requests", handleGetRequests)

	// Tumeifunga router yetu yote kwenye secureHeaders middleware
	securedHandler := secureHeaders(mux)

	http.ListenAndServe(":"+port, securedHandler)
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var u User
	_ = json.NewDecoder(r.Body).Decode(&u)

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

// handleLogin iliyorekebishwa: Inakagua kama mtumiaji yupo kwenye database kabla ya kumruhusu kuingia
func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var loginData LoginReq
	_ = json.NewDecoder(r.Body).Decode(&loginData)

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
	w.Header().Set("Content-Type", "application/json")
	var req ServiceReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Safisha majina ili yasikosekane kwenye Dashboard
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
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	list := users
	mu.Unlock()
	json.NewEncoder(w).Encode(list)
}

func handleGetRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	list := requests
	mu.Unlock()
	json.NewEncoder(w).Encode(list)
}
