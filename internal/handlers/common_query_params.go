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
			if value == "liked" {
				opts.Order = aws.Int(store.OrderByLiked)
			} else if value == "disliked" {
				opts.Order = aws.Int(store.OrderByDisliked)
			} else {
				opts.Order = aws.Int(store.OrderByBoth)
			}
		}
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
		str += fmt.Sprintf(`limit=%d`, *queryOpts.Limit)
	}
	if queryOpts.Order != nil {
		str += fmt.Sprintf(`order=%d`, *queryOpts.Order)
	}
	if queryOpts.Site != nil {
		str += fmt.Sprintf(`site=%s`, *queryOpts.Site)
	}
	return str
}
