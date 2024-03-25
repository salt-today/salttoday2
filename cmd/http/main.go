package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/salt-today/salttoday2/internal/handlers"
	"github.com/salt-today/salttoday2/internal/store"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	storage, err := store.NewSQLStorage(context.Background())
	if err != nil {
		panic(err)
	}

	handler := handlers.NewHandler(storage)

	// htmx
	r.Get("/", handler.HandleHome)
	r.Get("/comments", handler.HandleGetComments)

	r.Handle("/public/*", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	port := os.Getenv("PORT")
	if port == `` {
		port = "8080"
	}

	isDeployed := os.Getenv("RAILWAY_PUBLIC_DOMAIN") != ``
	domain := "localhost"
	if isDeployed {
		domain = ""
	}

	println("Listening on :", port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%s", domain, port), r)
	if err != nil {
		panic(err)
	}
}
