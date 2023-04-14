package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"
)

// TODO keep?...
func (s *httpService) GetUserCommentsHTTPHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var userIdString = chi.URLParam(r, "userID")
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Errorf("userId was not a valid number: %w", err).Error()))
		return
	}

	// Query values can in theory be repeated, but we won't support that, so squash em'
	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})
	opts, err := processGetCommentQueryParameters(params)
	opts.UserID = aws.Int(userId)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	opts.UserID = aws.Int(userId)

	userComments, err := s.storage.GetComments(ctx, *opts)
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
