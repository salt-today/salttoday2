package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/salt-today/salttoday2/internal/server/handlers"
	"github.com/salt-today/salttoday2/internal/store/rdb"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	storage, err := rdb.New(context.Background())
	if err != nil {
		panic(err)
	}

	handler := handlers.NewHandler(storage)

	// htmx
	// home/comments page
	r.Get("/", handler.HandleHome)
	r.Get("/comments", handler.HandleGetComments)

	// about page
	r.Get("/about", handler.HandleAbout)

	// user page
	r.Get("/user/{userID}", handler.HandleUser)
	r.Get("/user/{userID}/comments", handler.HandleGetUserComments)

	r.Handle("/public/*", http.StripPrefix("/public/", http.FileServer(http.Dir("web/public"))))

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
