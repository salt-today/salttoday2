package handlers

import (
	"net/http"

	"github.com/salt-today/salttoday2/internal/ui/views"
)

func (h *Handler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	views.About().Render(r.Context(), w)
}
