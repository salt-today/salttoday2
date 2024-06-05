package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/server/ui/components"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleCommentsPage(w http.ResponseWriter, r *http.Request) {
	entry := logger.New(r.Context()).WithField("handler", "Home")

	queryOpts, err := processGetCommentQueryParameters(r, 7)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	comments, err := h.storage.GetComments(r.Context(), queryOpts)
	if err != nil && !errors.Is(err, &store.NoQueryResultsError{}) {
		entry.WithError(err).Warn("error getting comments")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	nextUrl := getNextCommentsUrl(queryOpts)

	if queryOpts.PageOpts.Site != `` {
		comments = stripArticleSiteName(comments)
	} else {
		comments = stripArticleSiteNameAll(comments)
	}

	hxTrigger := r.Header.Get("HX-Trigger")
	if hxTrigger == "pagination" || hxTrigger == "form" {
		components.CommentsListComponent(comments, nextUrl).Render(r.Context(), w)
		return
	}

	views.Home(queryOpts, comments, getNextCommentsUrl(queryOpts), internal.SitesMapKeys).Render(r.Context(), w)
}

func (h *Handler) HandleComment(w http.ResponseWriter, r *http.Request) {
	entry := logger.New(r.Context()).WithField("handler", "GetComment")

	commentIDStr := chi.URLParam(r, "commentID")
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		entry.WithError(err).Warn("invalid comment id")
	}
	entry = entry.WithField("commentID", commentID)

	queryOpts := &store.CommentQueryOptions{
		ID:       &commentID,
		PageOpts: &store.PageQueryOptions{},
	}

	comments, err := h.storage.GetComments(r.Context(), queryOpts)
	if err != nil && !errors.Is(err, &store.NoQueryResultsError{}) {
		// TODO make this check a reusable func?
		entry.WithError(err).Warn("error getting comment")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	views.Comment(comments[0]).Render(r.Context(), w)
}

func stripArticleSiteNameAll(comments []*store.Comment) []*store.Comment {
	for _, comment := range comments {
		if comment.Article.SiteName == internal.AllSitesName {
			comment.Article.SiteName = ""
		}
	}
	return comments
}

func stripArticleSiteName(comments []*store.Comment) []*store.Comment {
	for _, comment := range comments {
		comment.Article.SiteName = ""
	}
	return comments
}

func processGetCommentQueryParameters(r *http.Request, defaultDays uint) (*store.CommentQueryOptions, error) {
	parameters := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})

	pageOpts, err := processPageQueryParams(parameters)
	if err != nil {
		return nil, err
	}

	opts := &store.CommentQueryOptions{
		PageOpts: pageOpts,
		DaysAgo:  defaultDays,
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
			daysAgo, err := ParseUint(value)
			if err != nil {
				return nil, fmt.Errorf("days_ago was not a valid number: %w", err)
			}
			opts.DaysAgo = daysAgo

		case "text":
			opts.Text = value
		}
	}

	return opts, nil
}

func getNextCommentsUrl(queryOpts *store.CommentQueryOptions) string {
	paramsString := ``
	path := `/`
	if queryOpts.UserID != nil {
		path = fmt.Sprintf(`/user/%d/`, *queryOpts.UserID)
	}
	if queryOpts.OnlyDeleted {
		paramsString += `&only_deleted=true`
	}
	paramsString += fmt.Sprintf(`&days_ago=%d`, queryOpts.DaysAgo)
	if queryOpts.Text != `` {
		paramsString += fmt.Sprintf(`&text=%s`, queryOpts.Text)
	}

	if len(paramsString) > 0 {
		paramsString = paramsString[1:]
	}

	nextPageParamsString := getNextPageQueryString(queryOpts.PageOpts)
	return fmt.Sprintf("%s?%s&%s", path, paramsString, nextPageParamsString)
}
