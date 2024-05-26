package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/server/ui/components"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleUserPage(w http.ResponseWriter, r *http.Request) {
	entry := logger.New(r.Context()).WithField("handler", "User")

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		entry.WithError(err).Warn("invalid user id")
	}
	entry = entry.WithField("userID", userID)

	userOpts, err := processGetUsersQueryParameters(r)
	if err != nil {
		entry.Error("error parsing user query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	userOpts.ID = &userID

	users, err := h.storage.GetUsers(r.Context(), userOpts)
	if errors.Is(err, &store.NoQueryResultsError{}) {
		entry.WithError(err).Warning("invalid user")
		w.WriteHeader(404)
		return
	} else if err != nil {
		entry.WithError(err).Error("unknown error getting user")
		w.WriteHeader(500)
		return
	}
	if len(users) < 1 {
		entry.Warning("invalid user")
		w.WriteHeader(404)
	}

	commentOpts, err := processGetCommentQueryParameters(r, 0)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	commentOpts.UserID = &userID

	comments, err := h.storage.GetComments(r.Context(), commentOpts)
	if err != nil && !errors.Is(err, &store.NoQueryResultsError{}) {
		entry.WithError(err).Warn("error getting comments")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	nextUrl := getNextCommentsUrl(commentOpts)

	if commentOpts.PageOpts.Site != `` {
		comments = stripArticleSiteName(comments)
	} else {
		comments = stripArticleSiteNameAll(comments)
	}

	hxTrigger := r.Header.Get("HX-Trigger")
	if hxTrigger == "pagination" || hxTrigger == "form" {
		components.CommentsListComponent(comments, nextUrl).Render(r.Context(), w)
		return
	}

	views.User(users[0], commentOpts, comments, getNextCommentsUrl(commentOpts), internal.SitesMapKeys).Render(r.Context(), w)
}
