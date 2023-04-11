package service

import "github.com/salt-today/salttoday2/internal/store"

type httpService struct {
	storage store.Storage
}

func NewService(storage store.Storage) *httpService {
	return &httpService{
		storage: storage,
	}
}
