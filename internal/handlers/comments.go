package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/ui/components"
	"github.com/salt-today/salttoday2/internal/ui/views"
	"github.com/samber/lo"
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
	entry.Info(queryOpts.PageOpts.Order)

	comments, err := h.storage.GetComments(r.Context(), *queryOpts)
	if errors.Is(err, &store.NoQueryResultsError{}) {
		components.NoResultsFound().Render(r.Context(), w)
		return
	} else if err != nil {
		entry.WithError(err).Warn("error getting comments")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	views.Home(comments, getNextCommentsUrl(queryOpts)).Render(r.Context(), w)
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
		if queryOpts.PageOpts.Page == nil || *queryOpts.PageOpts.Page == 0 {
			components.NoResultsFound().Render(r.Context(), w)
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

			if daysAgo > 0 {
				opts.DaysAgo = aws.Uint(uint(daysAgo))
			} else {
				// Not setting is All Time
			}
		}
	}
	return opts, nil
}

func getNextCommentsUrl(queryOpts *store.CommentQueryOptions) string {
	paramsString := ``
	if queryOpts.UserID != nil {
		paramsString += fmt.Sprintf(`&user_id=%d`, *queryOpts.UserID)
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
	return `/comments?` + paramsString + `&` + getNextPageQueryString(&queryOpts.PageOpts)
}
