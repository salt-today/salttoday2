package main

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/salt-today/salttoday2/internal/service"
	"github.com/salt-today/salttoday2/internal/store"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	storage, err := store.NewSQLStorage(context.Background())
	if err != nil {
		panic(err)
	}

	svc := service.NewService(storage)

	r.Get("/api/v1/users", svc.GetUsersHTTPHandler)
	r.Get("/api/v1/users/{userID}", svc.GetUserHTTPHandler)
	r.Get("/api/v1/users/{userID}/comments", svc.GetUserCommentsHTTPHandler)

	r.Get("/api/v1/comments", svc.GetCommentsHTTPHandler)
	r.Get("/api/v1/comments/{commentID}", svc.GetCommentHTTPHandler)

	println("Listening on :3000")
	err = http.ListenAndServe(":3000", r)
	if err != nil {
		panic(err)
	}
}
