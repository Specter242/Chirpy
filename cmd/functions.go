package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func respondWithError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(`{"status": "error", "message": "` + message + `"}`))
}

func respondWithJSON(w http.ResponseWriter, status int, data []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write(data)
}

func badWordReplace(input string) string {
	badWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}

	words := strings.Fields(input)
	for i, word := range words {
		lower := strings.ToLower(word)
		if _, found := badWords[lower]; found {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func (cfg *apiConfig) handleFileServerHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Add(1)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File server hit recorded"))
}

func (cfg *apiConfig) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	data, err := os.ReadFile("admin/metrics.html")
	if err != nil {
		http.Error(w, "Could not read metrics.html", http.StatusInternalServerError)
		return
	}
	html := strings.ReplaceAll(string(data), "{{hits}}", fmt.Sprintf("%d", cfg.fileserverHits.Load()))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func (cfg *apiConfig) handleResetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Metrics reset"))
}

func (cfg *apiConfig) handleValidateChirp(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
	}
	if len(body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}
	var req struct {
		Body string `json:"body"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}
	cleaned_body := badWordReplace(req.Body)
	jsonResponse := fmt.Sprintf(`{"valid": true, "cleaned_body": "%s"}`, cleaned_body)
	respondWithJSON(w, http.StatusOK, []byte(jsonResponse))
}
