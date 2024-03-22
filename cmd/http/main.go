package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/salt-today/salttoday2/internal/handlers"
	"github.com/salt-today/salttoday2/internal/store"
)

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "/public/favicon.ico")
}

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

	r.Handle("/public/*", http.StripPrefix("/public/", http.FileServer(http.Dir("public"))))

	// old
	// r.Get("/api/v1/users", svc.GetUsersHTTPHandler)
	// r.Get("/api/v1/users/{userID}", svc.GetUserHTTPHandler)
	// r.Get("/api/v1/users/{userID}/comments", svc.GetUserCommentsHTTPHandler)

	// r.Get("/api/v1/comments", svc.GetCommentsHTTPHandler)
	// r.Get("/api/v1/comments/{commentID}", svc.GetCommentHTTPHandler)

	println("Listening on :42069")
	err = http.ListenAndServe(":42069", r)
	if err != nil {
		panic(err)
	}
}
