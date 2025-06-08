package main

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/Specter242/Chirpy/cmd/internal/database"
	"github.com/google/uuid"
)

type Server struct {
	Addr    string
	Mux     *http.ServeMux
	Handler func(http.ResponseWriter, *http.Request)
}

type apiConfig struct {
	fileserverHits atomic.Int32
	dbqueries      *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
