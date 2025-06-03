package main

import (
	"net/http"
	"sync/atomic"
)

type Server struct {
	Addr    string
	Mux     *http.ServeMux
	Handler func(http.ResponseWriter, *http.Request)
}

type apiConfig struct {
	fileserverHits atomic.Int32
}
