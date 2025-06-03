package main

import (
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	// Register /healthz handler directly
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fileServer := http.StripPrefix("/app", http.FileServer(http.Dir("./app")))
	mux.Handle("/app/", fileServer)

	http.ListenAndServe(":8080", mux)
}
