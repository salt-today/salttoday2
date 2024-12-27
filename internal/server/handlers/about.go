package handlers

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func (h *Handler) HandleAboutPage(w http.ResponseWriter, r *http.Request) {
	entry := logger.New(r.Context()).WithField("handler", "About")

	// Get top scoring user
	userOpts := &store.UserQueryOptions{
		PageOpts: &store.PageQueryOptions{
			Limit: aws.Uint(1),
			Page:  aws.Uint(0),
			Order: store.OrderByBoth,
		},
	}

	users, err := h.storage.GetUsers(r.Context(), userOpts)
	if err != nil {
		entry.WithError(err).Warn("error getting top user for about")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}

	topUser := &store.User{ID: 5876, UserName: "Soojavu"}
	if len(users) >= 1 {
		topUser = users[0]
	}

	stats, err := h.storage.GetStats(r.Context())
	if err != nil {
		entry.WithError(err).Warn("error getting stats")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
	}
	printer := message.NewPrinter(language.English)

	views.About(printer, topUser, stats).Render(r.Context(), w)
}
