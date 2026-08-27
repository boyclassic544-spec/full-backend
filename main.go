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
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

type ServiceReq struct {
	FullName    string `json:"fullName"`
	PhoneNumber string `json:"phoneNumber"`
	Service     string `json:"service"`
	Message     string `json:"message"`
	CreatedAt   string `json:"createdAt"`
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
	if u.Email == "" || len(u.Password) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Jaza taarifa sahihi"})
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
	u, exists := users[creds.Email]
	mu.Unlock()
	if !exists || u.Password != creds.Password {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Umekosea taarifa au hujajisajili!"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Umefanikiwa kuingia!"})
}

func handleService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var req ServiceReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	
	// Weka tarehe ya sasa moja kwa moja isisome "Invalid Date"
	req.CreatedAt = time.Now().Format("02 Jan 2006, 15:04")

	mu.Lock()
	requests = append(requests, req)
	mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Ombi limepokelewa na kuhifadhiwa vizuri!"})
}

func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	mu.Lock()
	list := []User{}
	for _, u := range users {
		list = append(list, User{FullName: u.FullName, Email: u.Email, PhoneNumber: u.PhoneNumber})
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
