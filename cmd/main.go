package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Specter242/Chirpy/cmd/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		return
	}
	dbQueries := database.New(db)
	cfg := &apiConfig{}
	cfg.dbqueries = dbQueries
	cfg.fileserverHits.Store(0)
	cfg.platform = os.Getenv("PLATFORM")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data, err := os.ReadFile("admin/metrics.html")
		if err != nil {
			http.Error(w, "Could not read metrics.html", http.StatusInternalServerError)
			return
		}
		html := fmt.Sprintf(string(data), cfg.fileserverHits.Load())
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if cfg.platform != "dev" {
			respondWithError(w, http.StatusForbidden, "Reset is only allowed in development mode")
			return
		}
		cfg.fileserverHits.Store(0)

		if err := cfg.dbqueries.Reset(context.Background()); err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not reset metrics")
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Metrics reset"))
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
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

		now := time.Now().UTC()
		user := User{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Email:     req.Email,
		}

		resp, err := json.Marshal(user)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not marshal user")
			return
		}
		respondWithJSON(w, http.StatusCreated, resp)
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Body   string `json:"body"`
			UserID int32  `json:"user_id"`
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
		chirp, err := cfg.dbqueries.CreateChirp(context.Background(), database.CreateChirpParams{
			Body:   cleanedBody,
			UserID: req.UserID,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not create chirp")
			return
		}
		resp, err := json.Marshal(chirp)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Could not marshal chirp")
			return
		}
		respondWithJSON(w, http.StatusCreated, resp)
	})

	fileServer := http.StripPrefix("/app", http.FileServer(http.Dir("./app")))
	mux.Handle("/app/", cfg.middlewareMetricsInc(fileServer))

	http.ListenAndServe(":8080", mux)
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
