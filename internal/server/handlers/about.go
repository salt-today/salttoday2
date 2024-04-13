package handlers

import (
	"net/http"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleAboutPage(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "About")

	users, err := h.storage.GetTopUser(r.Context(), store.OrderByBoth)
	if err != nil {
		entry.WithError(err).Warn("error getting top user for about")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}

	views.About(users).Render(r.Context(), w)
}
