package main

import (
	"encoding/json"
	"net/http"
	"os"
)

type AuthRequest struct {
	Username string `json:"username"d
	Password string `json:"password"d
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

	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/register", handleRegister)

	http.ListenAndServe(":"+port, nil)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}
	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Username == "" || len(req.Password) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Umekosea taarifa au password fupi mno!"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Umeingia kwa mafanikio!"})
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != "POST" {
		http.Error(w, "Method haikubaliwi", http.StatusMethodNotAllowed)
		return
	}
	var req AuthRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || req.Username == "" || len(req.Password) < 6 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Usajili umeshindikana: Password lazima iwe na herufi 6 au zaidi!"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Akaunti imetengenezwa salama kabisa!"})
}
