package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/go-chi/chi/v5"

	"github.com/salt-today/salttoday2/pkg/sdk"
	"github.com/salt-today/salttoday2/pkg/server/internal/ui/components"
	"github.com/salt-today/salttoday2/pkg/server/internal/ui/views"
	"github.com/salt-today/salttoday2/pkg/store"
)

func (h *Handler) HandleUser(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "User")

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		entry.WithError(err).Warn("invalid user id")
	}
	entry = entry.WithField("userID", userID)

	user, err := h.storage.GetUserByID(r.Context(), userID)
	if err != nil {
		entry.WithError(err).Warning("invalid user")
		w.WriteHeader(404)
	}

	queryOpts, err := processGetCommentQueryParameters(r)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	queryOpts.UserID = aws.Int(userID)

	comments, err := h.storage.GetComments(r.Context(), *queryOpts)
	if err != nil && errors.Is(err, &store.NoQueryResultsError{}) {
		entry.WithError(err).Warn("error getting comments")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}
	views.User(user, queryOpts, comments, getNextCommentsUrl(queryOpts)).Render(r.Context(), w)
}

func (h *Handler) HandleGetUserComments(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "GetUserComments")

	queryOpts, err := processGetCommentQueryParameters(r)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	userIDStr := chi.URLParam(r, "userID")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		entry.WithError(err).Warn("invalid user id")
	}
	queryOpts.UserID = aws.Int(userID)

	comments, err := h.storage.GetComments(r.Context(), *queryOpts)
	if errors.Is(err, &store.NoQueryResultsError{}) {
		// error on first page? no comments
		if *queryOpts.PageOpts.Page == 0 {
			components.NoResultsFound("User").Render(r.Context(), w)
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
