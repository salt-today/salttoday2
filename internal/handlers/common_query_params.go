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
	return &opts, nil
}
