package service

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
)

func (s *httpService) GetCommentHTTPHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logEntry := sdk.Logger(ctx).WithField("path", r.URL.Path)

	// Query values can in theory be repeated, but we won't support that, so squash em'
	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})
	opts, err := processGetCommentQueryParameters(params)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	// In this particular endpoint, we only ever want a single comment
	opts.Limit = aws.Uint(1)

	// Overwrite any ID present in the query parameters, with what's in the path parameter
	var commentIdString = chi.URLParam(r, "commentID")
	commentId, err := strconv.Atoi(commentIdString)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Errorf("commentID was not a valid number: %w", err).Error()))
		return
	} else {
		opts.ID = aws.Int(commentId)
	}

	if opts.ID == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Errorf("no Id provided").Error()))
		return
	}

	comments, err := s.storage.GetComments(ctx, *opts)
	if err != nil {
		logEntry.WithError(err).Error("Failed to get comments")
		errorHandler(err, w, r)
		return
	}
	if len(comments) != 1 {
		w.WriteHeader(500)
		w.Write([]byte(fmt.Errorf("unexpected number of comments present: %d", len(comments)).Error()))
		return
	}

	if *opts.Format == "json" {
		responseBytes, err := json.Marshal(comments[0])
		if err != nil {
			logEntry.WithError(err).Error("Failed to create response")
			errorHandler(err, w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(responseBytes)
	} else if *opts.Format == "html" {
		// TODO Render an opengraph html snippet
		tmpl, err := template.ParseFiles("")
		if err != nil {
			logEntry.WithError(err).Error("Failed to get comments")
			errorHandler(err, w, r)
			return
		}

		err = tmpl.Execute(w, comments[0])
		if err != nil {
			logEntry.WithError(err).Error("Failed to get comments")
			errorHandler(err, w, r)
			return
		}
	}
}

func (s *httpService) GetCommentsHTTPHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logEntry := sdk.Logger(ctx).WithField("path", r.URL.Path)

	// Query values can in theory be repeated, but we won't support that, so squash em'
	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})
	opts, err := processGetCommentQueryParameters(params)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	// if username is set we need to get user id first
	if opts.UserName != nil {
		user, err := s.storage.GetUserByName(ctx, *opts.UserName)
		if err != nil {
			logEntry.WithError(err).Error("Failed to get user by name")
			errorHandler(err, w, r)
			return
		}
		opts.UserID = aws.Int(user.ID)
	}

	comments, err := s.storage.GetComments(ctx, *opts)
	if err != nil {
		logEntry.WithError(err).Error("Failed to get comments")
		errorHandler(err, w, r)
		return
	}

	userIDs := make([]int, len(comments))
	for i, comment := range comments {
		userIDs[i] = comment.UserID
	}
	users, err := s.storage.GetUsersByIDs(ctx, userIDs...)
	if err != nil {
		logEntry.WithError(err).Error("Failed to get users by ids")
		errorHandler(err, w, r)
		return
	}

	articleIDs := make([]int, len(comments))
	for i, comment := range comments {
		articleIDs[i] = comment.ArticleID
	}
	// TODO - for some reason this fails if we pass in 0 ids, but GetUserByIDs doesn't...
	articles, err := s.storage.GetArticles(ctx, articleIDs...)
	if err != nil {
		logEntry.WithError(err).Error("Failed to get articles by ids")
		errorHandler(err, w, r)
		return
	}

	responseBytes, err := createResponse(comments, users, articles)
	if err != nil {
		logEntry.WithError(err).Error("Failed to create response")
		errorHandler(err, w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func createResponse(comments []*store.Comment, users []*store.User, articles []*store.Article) ([]byte, error) {
	responseMap := map[string]interface{}{}
	responseMap["comments"] = comments

	usersMap := make(map[int]string)
	for _, user := range users {
		usersMap[user.ID] = user.UserName
	}
	responseMap["users"] = usersMap

	articlesMap := make(map[int]*store.Article)
	for _, article := range articles {
		articlesMap[article.ID] = article
	}
	responseMap["articles"] = articlesMap

	return json.Marshal(responseMap)
}

func processGetCommentQueryParameters(parameters map[string]string) (*store.CommentQueryOptions, error) {
	var opts store.CommentQueryOptions
	for param, value := range parameters {
		switch strings.ToLower(param) {
		case "id":
			commentId, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("id was not a valid number: %w", err)
			}
			opts.ID = aws.Int(commentId)
			break // it doesn't get much more specific than this, so we can stop here
		case "user_id":
			userId, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("user_id was not a valid number: %w", err)
			}
			opts.UserID = aws.Int(userId)
		case "user_name":
			opts.UserName = aws.String(value)
		case "limit":
			limitValue, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("limit was not a valid number: %w", err)
			}
			// TODO How to strconv uint?
			opts.Limit = aws.Uint(uint(limitValue))
		case "page":
			pageValue, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("page was not a valid number: %w", err)
			}
			opts.Page = aws.Uint(uint(pageValue))
		case "only_deleted":
			onlyDeletedValue, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("only_deleted was not a valid boolean: %w", err)
			}
			opts.OnlyDeleted = onlyDeletedValue
		case "order":
			if value == "liked" {
				opts.Order = aws.Int(store.OrderByLiked)
			} else if value == "disliked" {
				opts.Order = aws.Int(store.OrderByDisliked)
			} else {
				opts.Order = aws.Int(store.OrderByBoth)
			}
		case "format":
			if value == "html" {
				opts.Format = aws.String("html")
			} else if value == "json" {
				opts.Format = aws.String("json")
			} else if value == "" {
				opts.Format = aws.String("html")
			} else {
				return nil, fmt.Errorf("invalid format requested: %s", value)
			}
		case "days_ago":
			daysAgo, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("days_ago was not a valid number: %w", err)
			}
			opts.DaysAgo = aws.Uint(uint(daysAgo))
		}
	}
	return &opts, nil
}
