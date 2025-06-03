package main

import (
	"net/http"
)

func main() {
	server := &Server{
		Addr: ":8080",
		Mux:  http.NewServeMux(),
	}

	fileServer := http.FileServer(http.Dir("."))
	server.Mux.Handle("/", fileServer)

	http.ListenAndServe(server.Addr, server.Mux)
}
