package main

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

type User struct {
	FullName    string `json:"fullName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

var (
	mu    sync.Mutex
	users = make(map[string]User)
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

	http.ListenAndServe(":"+port, nil)
}

func handleSignup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Method not allowed"})
		return
	}

	var u User
	err := json.NewDecoder(r.Body).Decode(&u)
	if err != nil || u.Email == "" || len(u.Password) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Jaza taarifa sahihi au password iwe na herufi 6+"})
		return
	}

	mu.Lock()
	if _, exists := users[u.Email]; exists {
		mu.Unlock()
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Akaunti tayari ipo, nenda ukalogin!"})
		return
	}
	users[u.Email] = u
	mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Usajili umekamilika! Sasa unaweza kulogin."})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Method not allowed"})
		return
	}

	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Taarifa mbovu za login"})
		return
	}

	mu.Lock()
	u, exists := users[creds.Email]
	mu.Unlock()

	if !exists || u.Password != creds.Password {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Hujajisajili au umekosea password! Tafadhali create account kwanza."})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Umeingia kwa mafanikio!"})
}

func handleService(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Ombi limepokelewa!"})
}
