package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ServiceReq struct {
	FullName        string `json:"full_name"`
	Phone           string `json:"phone"`
	ServiceCategory string `json:"service_category"`
	Description     string `json:"description"`
	CreatedAt       string `json:"created_at"`
}

var (
	mu       sync.Mutex
	users    = make(map[string]User)
	requests = []ServiceReq{}
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "admin.html")
	})

	http.HandleFunc("/api/signup", handleSignup)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/service-requests", handleService)
	http.HandleFunc("/api/admin/users", handleGetUsers)
	http.HandleFunc("/api/admin/requests", handleGetRequests)

	http.ListenAndServe(":"+port, nil)
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var u User
	_ = json.NewDecoder(r.Body).Decode(&u)
	if u.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Jaza email"})
		return
	}
	mu.Lock()
	users[u.Email] = u
	mu.Unlock()
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Imefaulu!"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var creds User
	_ = json.NewDecoder(r.Body).Decode(&creds)
	mu.Lock()
	_, exists := users[creds.Email]
	mu.Unlock()
	if !exists {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Hujajisajili!"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Umefanikiwa!"})
}

func handleService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req ServiceReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	
	req.CreatedAt = time.Now().Format("02 Jan 2006, 15:04")

	mu.Lock()
	requests = append(requests, req)
	mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Imepokelewa!"})
}

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	list := []User{}
	for _, u := range users {
		list = append(list, u)
	}
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
