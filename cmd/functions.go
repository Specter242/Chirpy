package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/Specter242/Chirpy/cmd/internal/database"
	"github.com/google/uuid"
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
	if err := cfg.dbqueries.Reset(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not reset metrics")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Metrics reset"))
}

func (cfg *apiConfig) handleChirps(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Something went wrong")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}
	if len(req.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}
	cleanedBody := badWordReplace(req.Body)
	dbChirp, err := cfg.dbqueries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: req.UserID,
	})
	if err != nil {
		fmt.Println("Error creating chirp:", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create chirp")
		return
	}

	chirp := Chirp{
		ID:        dbChirp.ID,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
		CreatedAt: dbChirp.CreatedAt.Time,
		UpdatedAt: dbChirp.UpdatedAt.Time,
	}

	resp, err := json.Marshal(chirp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not marshal chirp")
		return
	}
	respondWithJSON(w, http.StatusCreated, resp)
}

func (cfg *apiConfig) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}
	if req.Email == "" {
		respondWithError(w, http.StatusBadRequest, "Email is required")
		return
	}

	dbUser, err := cfg.dbqueries.CreateUser(r.Context(), req.Email)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") {
			respondWithError(w, http.StatusConflict, "Email already exists")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Could not create user")
		return
	}

	user := User{
		ID:        dbUser.ID,
		Email:     dbUser.Email,
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}

	resp, err := json.Marshal(user)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not marshal user")
		return
	}
	respondWithJSON(w, http.StatusCreated, resp)
}

func (cfg *apiConfig) handleGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.dbqueries.GetAllChirps(r.Context(), database.GetAllChirpsParams{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not retrieve chirps")
		return
	}

	var response []Chirp
	for _, dbChirp := range chirps {
		response = append(response, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt.Time,
			UpdatedAt: dbChirp.UpdatedAt.Time,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		})
	}

	resp, err := json.Marshal(response)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not marshal chirps")
		return
	}
	respondWithJSON(w, http.StatusOK, resp)
}

func (cfg *apiConfig) handleGetChirpByID(w http.ResponseWriter, r *http.Request) {
	chirpID := strings.TrimPrefix(r.URL.Path, "/api/chirps/")
	if chirpID == "" {
		respondWithError(w, http.StatusBadRequest, "Chirp ID is required")
		return
	}

	id, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Chirp ID format")
		return
	}

	dbChirp, err := cfg.dbqueries.GetChirpByID(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Could not retrieve chirp")
		return
	}

	chirp := Chirp{
		ID:        dbChirp.ID,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
		CreatedAt: dbChirp.CreatedAt.Time,
		UpdatedAt: dbChirp.UpdatedAt.Time,
	}

	resp, err := json.Marshal(chirp)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not marshal chirp")
		return
	}
	respondWithJSON(w, http.StatusOK, resp)
}
