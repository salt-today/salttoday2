package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/samber/lo"

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

	topUser, err := h.storage.GetTopUser(r.Context(), *userOpts.PageOpts.Order)
	if err != nil {
		entry.WithError(err).Error("error getting top user")
		w.WriteHeader(500)
		return
	}

	nextUrl := getNextUsersUrl(userOpts)
	views.Users(users, topUser, userOpts, nextUrl).Render(r.Context(), w)
}

func (h *Handler) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	entry := sdk.Logger(r.Context()).WithField("handler", "GetUsers")

	userOpts, err := processGetUsersQueryParameters(r)
	if err != nil {
		entry.Error("error parsing query parameters", err)
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}

	users, err := h.storage.GetUsers(r.Context(), userOpts)

	if errors.Is(err, &store.NoQueryResultsError{}) {
		// error on first page? no comments
		if *userOpts.PageOpts.Page == 0 {
			components.NoResultsFound("users").Render(r.Context(), w)
			return
		}
		// end of comments - nothing to do here
		return
	} else if err != nil {
		entry.WithError(err).Warn("error getting users")
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	topUser, err := h.storage.GetTopUser(r.Context(), *userOpts.PageOpts.Order)
	if err != nil {
		entry.Error("error getting top user")
		w.WriteHeader(500)
		return
	}

	nextUrl := getNextUsersUrl(userOpts)
	components.UsersListComponent(users, *userOpts.PageOpts.Order, topUser, nextUrl).Render(r.Context(), w)

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
	path := `/api/users`

	nextPageParamsString := getNextPageQueryString(queryOpts.PageOpts)
	return fmt.Sprintf("%s?%s", path, nextPageParamsString)
}
