package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/server/ui/components"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleUserPage(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "User")

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		entry.WithError(err).Warn("invalid user id")
	}
	entry = entry.WithField("userID", userID)

	commentOpts, err := processGetCommentQueryParameters(r, 0)
	if err != nil {
		entry.Error("error parsing comment query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	commentOpts.UserID = &userID

	comments, err := h.storage.GetComments(r.Context(), commentOpts)
	if errors.Is(err, &store.NoQueryResultsError{}) {
		entry.WithError(err).Warn("error getting comments")
		w.WriteHeader(404)
		w.Write([]byte(err.Error()))
		return
	}

	hxTrigger := r.Header.Get("HX-Trigger")
	if hxTrigger == "pagination" || hxTrigger == "form" {
		components.CommentsListComponent(comments, getNextCommentsUrl(commentOpts)).Render(r.Context(), w)
		return
	}

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
	views.User(users[0], commentOpts, comments, getNextCommentsUrl(commentOpts)).Render(r.Context(), w)
}
