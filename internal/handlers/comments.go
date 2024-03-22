package handlers

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/ui/components"
	"github.com/salt-today/salttoday2/internal/ui/views"
)

func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "Home")

	queryOpts := getCommentQueryOptions(r)
	comments, err := h.storage.GetComments(r.Context(), queryOpts)
	if err != nil {
		entry.Error("error getting comments", err)
		w.WriteHeader(500)
	}
	entry.Info("Showing comments list!")
	views.Home(comments).Render(r.Context(), w)
}

func (h *Handler) HandleGetComments(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "GetComments")

	queryOpts := getCommentQueryOptions(r)
	comments, err := h.storage.GetComments(r.Context(), queryOpts)
	if err != nil {
		entry.Info("error getting comments")
		w.WriteHeader(500)
	}
	entry.Info("Showing comments list!")
	components.CommentsListComponent(comments).Render(r.Context(), w)
}

func getCommentQueryOptions(r *http.Request) store.CommentQueryOptions {
	return store.CommentQueryOptions{
		PageOpts: store.PageQueryOptions{
			Order: aws.Int(store.OrderByBoth),
		},
	}
}
