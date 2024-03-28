package store

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// var _ Storage = (*dynamodbStorage)(nil)

type dynamodbStorage struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamodbStorage(cfg aws.Config, tableName string) *dynamodbStorage {
	svc := dynamodb.NewFromConfig(cfg)
	return &dynamodbStorage{
		client:    svc,
		tableName: tableName,
	}
}

func (s *dynamodbStorage) AddComments(ctx context.Context, comments ...*Comment) error {
	var items []types.WriteRequest
	for _, c := range comments {
		item, err := attributevalue.MarshalMap(c)
		if err != nil {
			return err
		}

		items = append(items, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		})
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{s.tableName: items},
	})

	return err
}

func (s *dynamodbStorage) GetUserComments(ctx context.Context, userID int, opts *CommentQueryOptions) ([]*Comment, error) {
	var responseComments []*Comment

	keyEx := expression.KeyAnd(
		expression.Key("user_id").Equal(expression.Value(userID)),                      // PK
		expression.Key("time").LessThanEqual(expression.Value(time.Now().UnixMilli())), // SK
	)
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return responseComments, err
	}

	q, err := s.client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		Limit:                     aws.Int32(100),
	})
	if err != nil {
		return responseComments, err
	}

	err = attributevalue.UnmarshalListOfMaps(q.Items, &responseComments)
	return responseComments, err
}

func (s *dynamodbStorage) AddArticles(ctx context.Context, articles ...*Article) error {
	var items []types.WriteRequest
	for _, a := range articles {
		item, err := attributevalue.MarshalMap(a)
		if err != nil {
			return err
		}

		items = append(items, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		})
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{s.tableName: items},
	})

	return err
}

func (s *dynamodbStorage) AddUsers(ctx context.Context, users ...*User) error {
	var items []types.WriteRequest
	for _, u := range users {
		item, err := attributevalue.MarshalMap(u)
		if err != nil {
			return err
		}

		items = append(items, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: item,
			},
		})
	}

	_, err := s.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{s.tableName: items},
	})

	return err
}

func (s *dynamodbStorage) GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*Article, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *dynamodbStorage) SetArticleScrapedNow(ctx context.Context, articleIDs ...int) error {
	return fmt.Errorf("not yet implemented")
}

func (s *dynamodbStorage) SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error {
	return fmt.Errorf("not yet implemented")
}
