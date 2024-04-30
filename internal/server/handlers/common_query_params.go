package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/salt-today/salttoday2/internal/store"
)

func processPageQueryParams(parameters map[string]string) (*store.PageQueryOptions, error) {
	// defaults
	opts := &store.PageQueryOptions{
		Page:  aws.Uint(0),
		Order: aws.Int(store.OrderByBoth),
	}

	for param, value := range parameters {
		switch strings.ToLower(param) {
		case "limit":
			limitValue, err := ParseUint(value)
			if err != nil {
				return nil, fmt.Errorf("limit was not a valid number: %w", err)
			}

			opts.Limit = &limitValue
		case "page":
			pageValue, err := ParseUint(value)
			if err != nil {
				return nil, fmt.Errorf("page was not a valid number: %w", err)
			}
			opts.Page = &pageValue
		case "site":
			if value != "" {
				opts.Site = value
			}
		case "order":
			if value == "likes" {
				*opts.Order = store.OrderByLikes
			} else if value == "dislikes" {
				*opts.Order = store.OrderByDislikes
			} else {
				*opts.Order = store.OrderByBoth
			}
		}
	}

	return opts, nil
}

func getNextPageQueryString(queryOpts *store.PageQueryOptions) string {
	str := ``

	pageNum := 1
	if queryOpts.Page != nil {
		pageNum = int(*queryOpts.Page) + 1
	}
	str += fmt.Sprintf(`page=%d`, pageNum)

	if queryOpts.Limit != nil {
		str += fmt.Sprintf(`&limit=%d`, *queryOpts.Limit)
	}
	if queryOpts.Order != nil {
		order := "score"
		if *queryOpts.Order == store.OrderByLikes {
			order = "likes"
		} else if *queryOpts.Order == store.OrderByDislikes {
			order = "dislikes"
		}
		str += fmt.Sprintf(`&order=%s`, order)
	}
	if queryOpts.Site != `` {
		str += fmt.Sprintf(`&site=%s`, queryOpts.Site)
	}
	return str
}

func ParseUint(s string) (uint, error) {
	num, err := strconv.ParseUint(s, 10, 64)
	return uint(num), err
}
