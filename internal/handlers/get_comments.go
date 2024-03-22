package handlers

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"strconv"
// 	"strings"
//
// 	"github.com/aws/aws-sdk-go-v2/aws"
// 	"github.com/go-chi/chi/v5"
// 	"github.com/samber/lo"
//
// 	"github.com/salt-today/salttoday2/internal/sdk"
// 	"github.com/salt-today/salttoday2/internal/store"
// )
//
// func (s *httpService) GetCommentHTTPHandler(w http.ResponseWriter, r *http.Request) {
// 	ctx := r.Context()
// 	logEntry := sdk.Logger(ctx).WithField("path", r.URL.Path)
//
// 	// Query values can in theory be repeated, but we won't support that, so squash em'
// 	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
// 		return value[0]
// 	})
// 	opts, err := processGetCommentQueryParameters(params)
// 	if err != nil {
// 		w.WriteHeader(500)
// 		w.Write([]byte(err.Error()))
// 		return
// 	}
//
// 	// In this particular endpoint, we only ever want a single comment
// 	opts.PageOpts.Limit = aws.Uint(1)
//
// 	var commentIdString = chi.URLParam(r, "commentID")
// 	commentId, err := strconv.Atoi(commentIdString)
// 	if err != nil {
// 		w.WriteHeader(500)
// 		w.Write([]byte(fmt.Errorf("commentID was not a valid number").Error()))
// 		return
// 	}
// 	opts.ID = aws.Int(commentId)
//
// 	comments, err := s.storage.GetComments(ctx, *opts)
// 	if err != nil {
// 		logEntry.WithError(err).Error("Failed to get comments")
// 		errorHandler(err, w, r)
// 		return
// 	}
// 	if len(comments) != 1 {
// 		w.WriteHeader(500)
// 		w.Write([]byte(fmt.Errorf("unexpected number of comments present: %d", len(comments)).Error()))
// 		return
// 	}
//
// 	if *opts.Format == "json" {
// 		responseBytes, err := json.Marshal(comments[0])
// 		if err != nil {
// 			logEntry.WithError(err).Error("Failed to create response")
// 			errorHandler(err, w, r)
// 			return
// 		}
// 		w.WriteHeader(http.StatusOK)
// 		w.Write(responseBytes)
// 	} else if *opts.Format == "html" {
// 		err = s.commentPreviewTmpl.Execute(w, comments[0])
// 		if err != nil {
// 			logEntry.WithError(err).Error("Failed to get comments")
// 			errorHandler(err, w, r)
// 			return
// 		}
// 	}
// }
//
// func (s *httpService) GetCommentsHTTPHandler(w http.ResponseWriter, r *http.Request) {
// 	ctx := r.Context()
// 	logEntry := sdk.Logger(ctx).WithField("path", r.URL.Path)
//
// 	// Query values can in theory be repeated, but we won't support that, so squash em'
// 	params := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
// 		return value[0]
// 	})
// 	opts, err := processGetCommentQueryParameters(params)
// 	if err != nil {
// 		w.WriteHeader(500)
// 		w.Write([]byte(err.Error()))
// 		return
// 	}
//
// 	comments, err := s.storage.GetComments(ctx, *opts)
// 	if err != nil {
// 		logEntry.WithError(err).Error("Failed to get comments")
// 		errorHandler(err, w, r)
// 		return
// 	}
//
// 	userIDs := make([]int, len(comments))
// 	for i, comment := range comments {
// 		userIDs[i] = comment.UserID
// 	}
// 	users, err := s.storage.GetUsersByIDs(ctx, userIDs...)
// 	if err != nil {
// 		logEntry.WithError(err).Error("Failed to get users by ids")
// 		errorHandler(err, w, r)
// 		return
// 	}
//
// 	articleIDs := make([]int, len(comments))
// 	for i, comment := range comments {
// 		articleIDs[i] = comment.ArticleID
// 	}
// 	// TODO - for some reason this fails if we pass in 0 ids, but GetUserByIDs doesn't...
// 	articles, err := s.storage.GetArticles(ctx, articleIDs...)
// 	if err != nil {
// 		logEntry.WithError(err).Error("Failed to get articles by ids")
// 		errorHandler(err, w, r)
// 		return
// 	}
//
// 	responseBytes, err := createResponse(comments, users, articles)
// 	if err != nil {
// 		logEntry.WithError(err).Error("Failed to create response")
// 		errorHandler(err, w, r)
// 		return
// 	}
//
// 	w.WriteHeader(http.StatusOK)
// 	w.Write(responseBytes)
// }
//
// func createResponse(comments []*store.Comment, users []*store.User, articles []*store.Article) ([]byte, error) {
// 	responseMap := map[string]interface{}{}
// 	responseMap["comments"] = comments
//
// 	usersMap := make(map[int]string)
// 	for _, user := range users {
// 		usersMap[user.ID] = user.UserName
// 	}
// 	responseMap["users"] = usersMap
//
// 	articlesMap := make(map[int]*store.Article)
// 	for _, article := range articles {
// 		articlesMap[article.ID] = article
// 	}
// 	responseMap["articles"] = articlesMap
//
// 	return json.Marshal(responseMap)
// }
//
// func processGetCommentQueryParameters(parameters map[string]string) (*store.CommentQueryOptions, error) {
// 	pageOpts, err := processPageQueryParams(parameters)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	opts := &store.CommentQueryOptions{
// 		PageOpts: *pageOpts,
// 	}
//
// 	// TODO - should format be configured via Accept header instead?
// 	opts.Format = aws.String("html")
//
// 	for param, value := range parameters {
// 		switch strings.ToLower(param) {
// 		case "only_deleted":
// 			onlyDeletedValue, err := strconv.ParseBool(value)
// 			if err != nil {
// 				return nil, fmt.Errorf("only_deleted was not a valid boolean: %w", err)
// 			}
// 			opts.OnlyDeleted = onlyDeletedValue
// 		case "days_ago":
// 			daysAgo, err := strconv.Atoi(value)
// 			if err != nil {
// 				return nil, fmt.Errorf("days_ago was not a valid number: %w", err)
// 			}
// 			opts.DaysAgo = aws.Uint(uint(daysAgo))
// 		case "format":
// 			if value == "html" {
// 				opts.Format = aws.String("html")
// 			} else if value == "json" {
// 				opts.Format = aws.String("json")
// 			} else if value == "" {
// 				opts.Format = aws.String("html")
// 			} else {
// 				return nil, fmt.Errorf("invalid format requested: %s", value)
// 			}
// 		}
// 	}
// 	return opts, nil
// }
