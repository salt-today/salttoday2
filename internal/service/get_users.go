package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal/store"
)

func (s *httpService) GetUsersHTTPHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query values can in theory be repeated, but we won't support that, so squash em'
	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})
	opts, err := processGetUserQueryParameters(params)

	// In this particular endpoint, we only ever want a single comment
	opts.PageOpts.Limit = aws.Uint(1)

	var userIDString = chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDString)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Errorf("userID was not a valid number: %w", err).Error()))
		return
	}
	opts.ID = aws.Int(userID)

	users, err := s.storage.QueryUsers(ctx, *opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	responseBytes, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func (s *httpService) GetUserHTTPHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query values can in theory be repeated, but we won't support that, so squash em'
	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})
	opts, err := processGetUserQueryParameters(params)

	users, err := s.storage.QueryUsers(ctx, *opts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	responseBytes, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func processGetUserQueryParameters(parameters map[string]string) (*store.UserQueryOptions, error) {
	pageOpts, err := processPageQueryParams(parameters)
	if err != nil {
		return nil, err
	}

	opts := store.UserQueryOptions{
		PageOpts: *pageOpts,
	}
	for param, value := range parameters {
		switch strings.ToLower(param) {
		case "site":
			opts.Site = aws.String(value)
		}
	}
	return &opts, nil
}
