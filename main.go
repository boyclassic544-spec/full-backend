package main

import (
	"encoding/json"
	"net/http"
	"os"
)

type DataRequest struct {
	FullName    string `json:"fullName"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
	Username    string `json:"username"`
	Service     string `json:"service"`
	Message     string `json:"message"`
}

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

	http.HandleFunc("/api/login", handleApi)
	http.HandleFunc("/api/signup", handleApi)
	http.HandleFunc("/api/service-requests", handleApi)

	http.ListenAndServe(":"+port, nil)
}

func handleApi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Method not allowed"})
		return
	}
	
	var req DataRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Imepokelewa na kufanya kazi kikamilifu!"})
}
