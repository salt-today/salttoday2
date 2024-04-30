package handlers

import (
	"fmt"
	"net/http"

	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/server/ui/components"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleUsersPage(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "Users")

	userOpts, err := processGetUsersQueryParameters(r)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	users, err := h.storage.GetUsers(r.Context(), userOpts)
	if err != nil {
		entry.WithError(err).Warning("error listing users")
		w.WriteHeader(500)
		return
	}
	if len(users) < 1 {
		entry.Warning("no users found")
	}

	topUser, err := h.storage.GetTopUser(r.Context(), *userOpts.PageOpts.Order, userOpts.PageOpts.Site)
	if err != nil {
		entry.WithError(err).Error("error getting top user")
		w.WriteHeader(500)
		return
	}

	nextUrl := getNextUsersUrl(userOpts)

	hxTrigger := r.Header.Get("HX-Trigger")
	if hxTrigger == "pagination" || hxTrigger == "form" {
		components.UsersListComponent(users, *userOpts.PageOpts.Order, topUser, nextUrl).Render(r.Context(), w)
		return
	}

	views.Users(users, topUser, userOpts, internal.GetSites(), nextUrl).Render(r.Context(), w)
}

func processGetUsersQueryParameters(r *http.Request) (*store.UserQueryOptions, error) {
	parameters := lo.MapValues(r.URL.Query(), func(value []string, key string) string {
		return value[0]
	})

	pageOpts, err := processPageQueryParams(parameters)
	if err != nil {
		return nil, err
	}

	opts := &store.UserQueryOptions{
		PageOpts: pageOpts,
	}

	return opts, nil
}

func getNextUsersUrl(queryOpts *store.UserQueryOptions) string {
	path := `/users`

	nextPageParamsString := getNextPageQueryString(queryOpts.PageOpts)
	return fmt.Sprintf("%s?%s", path, nextPageParamsString)
}
