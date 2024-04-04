package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/server/ui/components"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "Home")

	queryOpts, err := processGetCommentQueryParameters(r)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	comments, err := h.storage.GetComments(r.Context(), *queryOpts)
	if err != nil && errors.Is(err, &store.NoQueryResultsError{}) {
		entry.WithError(err).Warn("error getting comments")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	views.Home(queryOpts, comments, getNextCommentsUrl(queryOpts)).Render(r.Context(), w)
}

func (h *Handler) HandleGetComments(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "GetComments")

	queryOpts, err := processGetCommentQueryParameters(r)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	comments, err := h.storage.GetComments(r.Context(), *queryOpts)
	if errors.Is(err, &store.NoQueryResultsError{}) {
		// error on first page? no comments
		if *queryOpts.PageOpts.Page == 0 {
			components.NoResultsFound("Comment").Render(r.Context(), w)
			return
		}
		// end of comments - nothing to do here
		return
	} else if err != nil {
		entry.WithError(err).Warn("error getting comments")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	components.CommentsListComponent(comments, getNextCommentsUrl(queryOpts)).Render(r.Context(), w)
}

func (h *Handler) HandleGetComment(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "GetComment")

	commentIDStr := chi.URLParam(r, "commentID")
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		entry.WithError(err).Warn("invalid comment id")
	}
	entry = entry.WithField("commentID", commentID)

	queryOpts := store.CommentQueryOptions{
		ID: aws.Int(commentID),
	}

	comments, err := h.storage.GetComments(r.Context(), queryOpts)

	views.Comment(comments[0]).Render(r.Context(), w)
}

func processGetCommentQueryParameters(r *http.Request) (*store.CommentQueryOptions, error) {
	parameters := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})

	pageOpts, err := processPageQueryParams(parameters)
	if err != nil {
		return nil, err
	}

	opts := &store.CommentQueryOptions{
		PageOpts: *pageOpts,
		DaysAgo:  aws.Uint(7),
	}

	for param, value := range parameters {
		switch strings.ToLower(param) {
		case "only_deleted":
			onlyDeletedValue, err := strconv.ParseBool(value)
			if err != nil {
				return nil, fmt.Errorf("only_deleted was not a valid boolean: %w", err)
			}
			if onlyDeletedValue {
				opts.OnlyDeleted = onlyDeletedValue
			}

		case "days_ago":
			daysAgo, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("days_ago was not a valid number: %w", err)
			}
			opts.DaysAgo = aws.Uint(uint(daysAgo))
		}

	}

	return opts, nil
}

func getNextCommentsUrl(queryOpts *store.CommentQueryOptions) string {
	paramsString := ``
	path := `/comments`
	if queryOpts.UserID != nil {
		path = fmt.Sprintf(`/user/%d/comments`, *queryOpts.UserID)
	}
	if queryOpts.OnlyDeleted {
		paramsString += `&only_deleted=true`
	}
	if queryOpts.DaysAgo != nil {
		paramsString += fmt.Sprintf(`&days_ago=%d`, *queryOpts.DaysAgo)
	}

	if len(paramsString) > 0 {
		paramsString = paramsString[1:]
	}

	nextPageParamsString := getNextPageQueryString(&queryOpts.PageOpts)
	return fmt.Sprintf("%s?%s&%s", path, paramsString, nextPageParamsString)
}
