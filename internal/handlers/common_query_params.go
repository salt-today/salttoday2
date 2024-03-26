package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/salt-today/salttoday2/internal/store"
)

func processPageQueryParams(parameters map[string]string) (*store.PageQueryOptions, error) {
	var opts store.PageQueryOptions
	for param, value := range parameters {
		switch strings.ToLower(param) {
		case "limit":
			limitValue, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("limit was not a valid number: %w", err)
			}

			opts.Limit = aws.Uint(uint(limitValue))
		case "page":
			pageValue, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("page was not a valid number: %w", err)
			}
			opts.Page = aws.Uint(uint(pageValue))
		case "site":
			opts.Site = aws.String(value)
		case "order":
			if value == "likes" {
				opts.Order = aws.Int(store.OrderByLikes)
			} else if value == "dislikes" {
				opts.Order = aws.Int(store.OrderByDislikes)
			} else {
				opts.Order = aws.Int(store.OrderByBoth)
			}
		}
	}
	if opts.Page == nil {
		opts.Page = aws.Uint(0)
	}
	if opts.Order == nil {
		opts.Order = aws.Int(store.OrderByBoth)
	}

	return &opts, nil
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
	if queryOpts.Site != nil {
		str += fmt.Sprintf(`&site=%s`, *queryOpts.Site)
	}
	return str
}
