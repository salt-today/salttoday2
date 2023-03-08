package api

import (
	"context"
	"net/http"

	"github.com/salt-today/salttoday2/internal/store"
)

const (
	somethingWentWrong = "Something went wrong"
	notFound = "Not found"
)

func errorHandler(err error, w http.ResponseWriter, r *http.Request) {
	if _, ok := err.(store.NoQueryResultsError); ok {
		w.WriteHeader(404)
		w.Write([]byte(notFound))
		return
	}
	w.WriteHeader(500)
	w.Write([]byte(somethingWentWrong))
}