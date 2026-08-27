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

var (
	mu       sync.Mutex
	users    = []User{}
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
	
	if u.FullName == "" {
		if u.Name != "" { u.FullName = u.Name } else { u.FullName = "Mteja Aliyejisajili" }
	}
	if u.Email == "" { u.Email = "haijulikani@domain.com" }

	mu.Lock()
	users = append(users, u)
	mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Imefaulu!"})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Umefanikiwa!"})
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
		if req.PhoneNumber != "" { req.Phone = req.PhoneNumber } else { req.Phone = "Namba haipo" }
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
