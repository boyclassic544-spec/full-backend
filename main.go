package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// Muundo wa data inayopokelewa kutoka kwa mteja (Frontend)
type ServiceRequest struct {
	FullName        string `json:"full_name"`
	Phone           string `json:"phone"`
	ServiceCategory string `json:"service_category"`
	Description     string `json:"description"`
}

func main() {
	// Weka port ya kusikiliza maombi (Railway inatumia PORT environment variable)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Route ya kupokea maombi ya huduma
	http.HandleFunc("/api/request-service", handleServiceRequest)

	// Route ya kawaida ya kupima kama server ipo hai
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Mfumo wa Back-end unafanya kazi vizuri kabisa!")
	})

	log.Printf("Server inaanza kusikiliza kwenye port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Imeshindwa kuanzisha server: ", err)
	}
}

func handleServiceRequest(w http.ResponseWriter, r *http.Request) {
	// Ruhusu CORS kwa ajili ya frontend yako
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method haijasemwa vizuri, tumia POST", http.StatusMethodNotAllowed)
		return
	}

	var req ServiceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Imeshindwa kusoma data zilizotumwa: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Tuma email notification kupitia Resend API
	go sendEmailNotification(req)

	// Mjulishe mteja kuwa ombi limepokelewa
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w.Encode(map[string]string{
		"status":  "success",
		"message": "Ombi lako limepokelewa kikamilifu na limetumwa kwa uongozi!",
	}))
}

func sendEmailNotification(req ServiceRequest) {
	// Inachukua Resend API Key kwa usalama kutoka kwenye System Environment
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Println("Hitilafu: RESEND_API_KEY haijawekwa kwenye environment variables!")
		return
	}

	emailContent := "Ombi Jipya la Huduma - Mfumo Salama\n\n" +
		"Jina la Mteja: " + req.FullName + "\n" +
		"Namba ya Simu: " + req.Phone + "\n" +
		"Aina ya Huduma: " + req.ServiceCategory + "\n" +
		"Maelezo ya Ziada: " + req.Description

	payload := map[string]interface{}{
		"from":    "onboarding@resend.dev",
		"to":      []string{"Khalidtamimu622@gmail.com"},
		"subject": "Ombi Jipya la Huduma - TechSecure",
		"text":    emailContent,
	}

	jsonPayload, _ := json.Marshal(payload)

	httpReq, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Println("Imeshindwa kuandaa ombi la HTTP la Resend:", err)
		return
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Println("Hitilafu wakati wa kutuma barua pepe kupitia Resend:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Println("Barua pepe imetumwa kwa mafanikio makubwa kupitia Resend API!")
	} else {
		log.Println("Resend imerudisha hitilafu, Status code:", resp.StatusCode)
	}
}
