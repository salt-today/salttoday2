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
	r.Get("/", handler.HandleCommentsPage)
	r.Get("/comments", handler.HandleCommentsPage)

	// individual comment page
	r.Get("/comment/{commentID}", handler.HandleComment)

	// about page
	r.Get("/about", handler.HandleAboutPage)

	// user leaderboard
	r.Get("/users", handler.HandleUsersPage)

	// user page
	r.Get("/user/{userID}", handler.HandleUserPage)
	r.Get("/user/{userID}/comments", handler.HandleUserPage)

	r.Handle("/public/*", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	isDeployed := os.Getenv("RAILWAY_PUBLIC_DOMAIN") != ``
	domain := "localhost"
	if isDeployed {
		domain = ""
	}

	println("Listening on: 8080")
	err = http.ListenAndServe(fmt.Sprintf("%s:8080", domain), r)
	if err != nil {
		panic(err)
	}
}
