package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal/store"
)

// TODO keep?...
func GetUserCommentsHTTPHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// TODO should we be making storage every time?
	storage, err := store.NewSQLStorage(ctx)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	var userIdString = chi.URLParam(r, "user_id")
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Errorf("userId was not a valid number: %w", err).Error()))

	}

	// Query values can in theory be repeated, but we won't support that, so squash em'
	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})
	opts, err := processQueryParameters(params)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}

	userComments, err := storage.GetUserComments(ctx, userId, *opts)
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	responseBytes, err := json.Marshal(userComments)
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func processQueryParameters(parameters map[string]string) (*store.QueryOptions, error) {
	var opts store.QueryOptions
	for param, value := range parameters {
		switch strings.ToLower(param) {
		case "limit":
			limitValue, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("limit was not a valid number: %w", err)
			}
			// TODO How to strconv uint?
			opts.Limit = aws.Uint(uint(limitValue))
		case "show_deleted":
			showDeletedValue, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("show_deleted was not a valid boolean: %w", err)
			}
			opts.ShowDeleted = aws.Bool(showDeletedValue)
		case "order":
			if value == "liked" {
				opts.Order = aws.Int(store.OrderByLiked)
			} else if value == "disliked" {
				opts.Order = aws.Int(store.OrderByDisliked)
			} else {
				return nil, fmt.Errorf("order was not a valid option, must be liked,disliked")
			}
		}
	}
	return &opts, nil
}
