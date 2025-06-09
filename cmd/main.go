package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/Specter242/Chirpy/cmd/internal/database"
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

	mux.HandleFunc("GET /admin/metrics", cfg.handleMetrics)

	mux.HandleFunc("POST /admin/reset", cfg.handleResetMetrics)

	mux.HandleFunc("POST /api/users", cfg.handleCreateUser)

	mux.HandleFunc("POST /api/chirps", cfg.handleChirps)

	mux.HandleFunc("GET /api/chirps", cfg.handleGetChirps)

	mux.HandleFunc("GET /api/chirps/{id}", cfg.handleGetChirpByID)

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
