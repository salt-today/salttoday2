package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/samber/lo"

	"github.com/salt-today/salttoday2/internal"
	"github.com/salt-today/salttoday2/internal/logger"
	"github.com/salt-today/salttoday2/internal/server/ui/components"
	"github.com/salt-today/salttoday2/internal/server/ui/views"
	"github.com/salt-today/salttoday2/internal/store"
)

func (h *Handler) HandleUsersPage(w http.ResponseWriter, r *http.Request) {
	entry := logger.New(r.Context()).WithField("handler", "Users")

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

	nextUrl := getNextUsersUrl(userOpts)

	// need the top user to correctly show bar graph on page
	userOpts.PageOpts.Limit = aws.Uint(1)
	userOpts.PageOpts.Page = aws.Uint(0)
	topUserArr, err := h.storage.GetUsers(r.Context(), userOpts)
	var topUser *store.User
	if err != nil {
		entry.WithError(err).Error("error getting top user")
		w.WriteHeader(500)
		return
	} else if len(topUserArr) < 1 {
		entry.Error("no top user found")
	} else {
		topUser = topUserArr[0]
	}

	hxTrigger := r.Header.Get("HX-Trigger")
	if hxTrigger == "pagination" || hxTrigger == "form" {
		components.UsersListComponent(users, *userOpts.PageOpts.Order, topUser, nextUrl).Render(r.Context(), w)
		return
	}

	views.Users(users, topUser, userOpts, internal.SitesMapKeys, nextUrl).Render(r.Context(), w)
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

	for param, value := range parameters {
		switch strings.ToLower(param) {
		case "name":
			opts.Name = value
		}
	}

	return opts, nil
}

func getNextUsersUrl(queryOpts *store.UserQueryOptions) string {
	path := `/users`
	paramsString := ``

	if queryOpts.Name != `` {
		paramsString += fmt.Sprintf(`&name=%s&`, queryOpts.Name)
	}

	paramsString += getNextPageQueryString(queryOpts.PageOpts)
	return fmt.Sprintf("%s?%s", path, paramsString)
}
