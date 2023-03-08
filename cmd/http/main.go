package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/salt-today/salttoday2/internal/api"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/api/v1/users/{userID}/comments", api.GetUserCommentsHTTPHandler)
	r.Get("/api/v1/comments", api.GetCommentsHTTPHandler)

	http.ListenAndServe(":3000", r)
}
