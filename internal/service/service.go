package service

import (
	"text/template"

	"github.com/salt-today/salttoday2/internal/store"
)

type httpService struct {
	storage            store.Storage
	commentPreviewTmpl *template.Template
}

func NewService(storage store.Storage) *httpService {
	return &httpService{
		storage:            storage,
		commentPreviewTmpl: template.Must(template.ParseFiles("templates/comment.html")),
	}
}
