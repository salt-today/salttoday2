package handlers

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "About")
	queryOpts := &store.UserQueryOptions{
		PageOpts: store.PageQueryOptions{
			Limit: aws.Uint(1),
			Order: aws.Int(store.OrderByBoth),
		},
	}

	users, err := h.storage.GetUsers(r.Context(), queryOpts)
	if err != nil {
		entry.WithError(err).Warn("error getting top user for about")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}

	views.About(users[0]).Render(r.Context(), w)
}
