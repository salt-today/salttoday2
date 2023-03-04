package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/salt-today/salttoday2/internal/lambdas/api"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/users/{user_id}", api.GetUserCommentsHTTPHandler)

	http.ListenAndServe(":3000", r)
}
