package main

import (
	"net/http"
)

type Server struct {
	Addr    string
	Mux     *http.ServeMux
	Handler func(http.ResponseWriter, *http.Request)
}
