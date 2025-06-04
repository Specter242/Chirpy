package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	cfg := &apiConfig{}

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
		cfg.fileserverHits.Store(0)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Metrics reset"))
	})

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Something went wrong")
			return
		}
		if len(body) > 140 {
			respondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}
		respondWithJSON(w, http.StatusOK, []byte(`{"valid": true}`))
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
