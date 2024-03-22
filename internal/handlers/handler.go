package handlers

import (
	"github.com/salt-today/salttoday2/internal/store"
)

type Handler struct {
	storage store.Storage
}

func NewHandler(storage store.Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}
