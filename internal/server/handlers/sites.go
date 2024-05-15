package handlers

import (
	"net/http"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/server/ui/components"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/samber/lo"
)

func (h *Handler) HandleSitesPage(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "Sites")

	parameters := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})

	queryOpts, err := processPageQueryParams(parameters)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	users, err := h.storage.GetSites(r.Context(), queryOpts)
	if err != nil {
		entry.WithError(err).Warning("error listing sites")
		w.WriteHeader(500)
		return
	}
	if len(users) < 1 {
		entry.Warning("no sites found")
	}

	topSite, err := h.storage.GetTopSite(r.Context(), *queryOpts.Order)
	if err != nil {
		entry.WithError(err).Error("error getting top site")
		w.WriteHeader(500)
		return
	}

	nextUrl := `/sites?` + getNextPageQueryString(queryOpts)

	hxTrigger := r.Header.Get("HX-Trigger")
	if hxTrigger == "pagination" || hxTrigger == "form" {
		components.SitesListComponent(users, *queryOpts.Order, topSite, nextUrl).Render(r.Context(), w)
		return
	}

	views.Sites(users, topSite, queryOpts, nextUrl).Render(r.Context(), w)
}
