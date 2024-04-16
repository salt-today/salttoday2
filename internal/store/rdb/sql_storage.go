package rdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"

	"github.com/salt-today/salttoday2/internal/sdk"
	"github.com/salt-today/salttoday2/internal/store"
	"github.com/salt-today/salttoday2/internal/store/rdb/migrations"
)

var _ store.Storage = (*sqlStorage)(nil)

type sqlStorage struct {
	db      *sql.DB
	dialect goqu.DialectWrapper

	cachedResults *cachedResults
}

// Store expensive reults here
type cachedResults struct {
	topScoringUser  *store.User
	topLikedUser    *store.User
	topDislikedUser *store.User
}

const maxPageSize uint = 20

func getSqlConnString(ctx context.Context) string {
	url := os.Getenv("MYSQL_URL")

	if url == `` {
		sdk.Logger(ctx).Info("Missing database configuration, defaulting to local dev")
		return "root:salt@tcp(localhost:3306)/salt"
	}
	return url
}

func New(ctx context.Context) (*sqlStorage, error) {
	entry := logrus.WithField(`component`, `sql-storage`)

	db, err := sql.Open("mysql", getSqlConnString(ctx)+"?parseTime=true")
	if err != nil {
		return nil, err
	}
	entry.Info("successfully connected to database")

	err = migrations.MigrateDb(db)
	if err != nil {
		return nil, err
	}

	s := &sqlStorage{
		db:            db,
		dialect:       goqu.Dialect("mysql"),
		cachedResults: &cachedResults{},
	}

	// periodically calculate the most likes, dislikes
	s.cacheTopUsers(ctx)
	ticker := time.NewTicker(time.Hour)
	go func() error {
		<-ticker.C
		return s.cacheTopUsers(ctx)
	}()

	go func() {
		<-ctx.Done()
		err := s.shutdown()
		if err != nil {
			sdk.Logger(ctx).WithError(err).Error("Error shutting down SQL storage")
		}
	}()

	return s, nil
}

func (s *sqlStorage) GetTopUser(ctx context.Context, orderBy int) (*store.User, error) {
	if orderBy == store.OrderByBoth {
		if s.cachedResults.topScoringUser == nil {
			return nil, &store.NoQueryResultsError{}
		}
		return s.cachedResults.topScoringUser, nil

	} else if orderBy == store.OrderByLikes {
		if s.cachedResults.topLikedUser == nil {
			return nil, &store.NoQueryResultsError{}
		}
		return s.cachedResults.topLikedUser, nil

	} else if orderBy == store.OrderByDislikes {
		if s.cachedResults.topDislikedUser == nil {
			return nil, &store.NoQueryResultsError{}
		}

		return s.cachedResults.topDislikedUser, nil

	} else {
		return nil, fmt.Errorf("unknown orderBy %d", orderBy)
	}
}

func (s *sqlStorage) cacheTopUsers(ctx context.Context) error {
	entry := sdk.Logger(ctx).WithField("cache", "top users")
	opts := &store.UserQueryOptions{
		PageOpts: &store.PageQueryOptions{
			Order: aws.Int(store.OrderByBoth),
			Limit: aws.Uint(1),
		},
	}

	users, err := s.GetUsers(ctx, opts)
	if err != nil || len(users) < 1 {
		entry.WithError(err).Error("unable to calculate highest scoring user")
	}
	s.cachedResults.topScoringUser = users[0]

	opts.PageOpts.Order = aws.Int(store.OrderByLikes)
	users, err = s.GetUsers(ctx, opts)
	if err != nil || len(users) < 1 {
		entry.WithError(err).Error("unable to calculate highest liked user")
	}
	s.cachedResults.topLikedUser = users[0]

	opts.PageOpts.Order = aws.Int(store.OrderByDislikes)
	users, err = s.GetUsers(ctx, opts)
	if err != nil || len(users) < 1 {
		entry.WithError(err).Error("unable to calculate highest disliked user")
		return err
	}
	s.cachedResults.topDislikedUser = users[0]

	entry.Info("successfully updated top users cache")
	return nil
}

func (s *sqlStorage) AddComments(ctx context.Context, comments []*store.Comment) error {
	// We need to add comments aritlce by article so we can easily determine if a comment was deleted or not
	articleCommentsMap := make(map[int][]*store.Comment)
	for _, comment := range comments {
		articleCommentsMap[comment.Article.ID] = append(articleCommentsMap[comment.Article.ID], comment)
	}

	var err error
	for articleID, comments := range articleCommentsMap {
		addErr := s.addCommentsToArticle(ctx, articleID, comments)
		err = errors.Join(err, addErr)
	}

	return err
}

func (s *sqlStorage) addCommentsToArticle(ctx context.Context, articleID int, comments []*store.Comment) error {
	entry := sdk.Logger(ctx).WithField("articleID", articleID)

	// Determine if any comments were deleted
	queryOpts := &store.CommentQueryOptions{
		ArticleID: &articleID,
		PageOpts:  &store.PageQueryOptions{},
	}

	storedComments, err := s.GetComments(ctx, queryOpts)
	if errors.Is(err, &store.NoQueryResultsError{}) {
		// no-op
		entry.Info("New article, no comments found")
	} else if err != nil {
		entry.WithError(err).Error("Unable to get comments while adding new comments, required for determining if comments are deleted")
		return err
	}

	commentsMap := make(map[int]*store.Comment)
	for _, comment := range comments {
		commentsMap[comment.ID] = comment
	}

	for _, storedComment := range storedComments {
		if _, ok := commentsMap[storedComment.ID]; !ok {
			entry.WithField("commentID", storedComment.ID).Info("Found comment was deleted!")
			storedComment.Deleted = true
			comments = append(comments, storedComment)
		} else {
			commentsMap[storedComment.ID].Deleted = false
		}
	}

	ds := s.dialect.Insert(CommentsTable).
		Cols(CommentsID, CommentsArticleID, CommentsUserID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes, CommentsDeleted).
		As(NewAlias).
		OnConflict(goqu.DoUpdate(CommentsLikes, goqu.C(LikesSuffix).Set(goqu.I(NewAliasLikes)))).
		OnConflict(goqu.DoUpdate(CommentsDislikes, goqu.C(DislikesSuffix).Set(goqu.I(NewAliasDislikes))))

	for _, comment := range comments {
		ds = ds.Vals(goqu.Vals{comment.ID, comment.Article.ID, comment.User.ID, comment.Time.Truncate(time.Second), comment.Text, comment.Likes, comment.Dislikes, comment.Deleted})
	}
	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) GetUsers(ctx context.Context, opts *store.UserQueryOptions) ([]*store.User, error) {
	sd := s.dialect.
		From(UsersTable).
		InnerJoin(goqu.T(CommentsTable).As(CommentsTable), goqu.On(goqu.I(UsersID).Eq(goqu.I(CommentsUserID)))).
		GroupBy(UsersID)

	// only get the comments we need since we're summing all the values
	cols := []interface{}{UsersID, UsersName}
	// TODO this feels kinda bad - can probably rework this.
	if opts.PageOpts.Order == nil || *opts.PageOpts.Order == store.OrderByLikes {
		cols = append(cols, goqu.SUM(CommentsLikes).As(UserLikes))
		sd = sd.Order(goqu.I(UserLikes).Desc())
	} else if *opts.PageOpts.Order == store.OrderByDislikes {
		cols = append(cols, goqu.SUM(CommentsDislikes).As(UserDislikes))
		sd = sd.Order(goqu.I(UserDislikes).Desc())
	} else {
		cols = append(cols, goqu.SUM(CommentsLikes).As(UserLikes), goqu.SUM(CommentsDislikes).As(UserDislikes))
		sd = sd.Order(goqu.L(UserLikes + "+ 2 * " + UserDislikes).Desc())
	}
	sd = sd.Select(cols...)

	if opts.ID != nil {
		sd = sd.Where(goqu.Ex{UsersID: opts.ID})
	}

	sd = addPaging(sd, opts.PageOpts)

	// TODO Implement other options
	// site

	sd = addPaging(sd, opts.PageOpts)

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*store.User, 0)

	for rows.Next() {
		u := &store.User{}
		dests := []interface{}{&u.ID, &u.UserName}
		if *opts.PageOpts.Order == store.OrderByLikes {
			dests = append(dests, &u.TotalLikes)
		} else if *opts.PageOpts.Order == store.OrderByDislikes {
			dests = append(dests, &u.TotalDislikes)
		} else {
			dests = append(dests, &u.TotalLikes, &u.TotalDislikes)
		}
		err := rows.Scan(dests...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user record: %w", err)
		}

		// TODO feels bad.
		// Have to calculate score since it's not calculated in select anymore
		u.TotalScore = 2*u.TotalDislikes + u.TotalLikes
		users = append(users, u)
	}
	fmt.Println(users)
	return users, nil
}

func (s *sqlStorage) GetComments(ctx context.Context, opts *store.CommentQueryOptions) ([]*store.Comment, error) {
	sd := s.dialect.
		Select(CommentsID, CommentsTime, CommentsText, CommentsLikes, CommentsDislikes, goqu.L(CommentsLikes+" + 2 * "+CommentsDislikes).
			As(CommentsScore), CommentsDeleted, CommentsArticleID, CommentsUserID, ArticlesID, ArticlesTitle, ArticlesUrl, UsersID, UsersName).
		From(CommentsTable).
		InnerJoin(goqu.T(UsersTable).As(UsersTable), goqu.On(goqu.I(CommentsUserID).Eq(goqu.I(UsersID)))).
		InnerJoin(goqu.T(ArticlesTable).As(ArticlesTable), goqu.On(goqu.I(CommentsArticleID).Eq(goqu.I(ArticlesID))))

	if opts.ID != nil {
		sd = sd.Where(goqu.Ex{CommentsID: opts.ID})
	}

	if opts.UserID != nil {
		sd = sd.Where(goqu.Ex{CommentsUserID: opts.UserID})
	}

	if opts.PageOpts.Site != nil {
		// TODO
		// sd = sd.Where()
	}

	if opts.OnlyDeleted == true {
		sd = sd.Where(goqu.Ex{CommentsDeleted: true})
	}

	if opts.DaysAgo != nil && *opts.DaysAgo != 0 {
		sd = sd.Where(goqu.I(CommentsTime).Gt(goqu.L("NOW() - INTERVAL ? DAY", *opts.DaysAgo)))
	}

	if opts.ArticleID != nil {
		sd = sd.Where(goqu.Ex{CommentsArticleID: *opts.ArticleID})
	}

	sd = addPaging(sd, opts.PageOpts)

	if opts.PageOpts.Order != nil {
		if *opts.PageOpts.Order == store.OrderByLikes {
			sd = sd.Order(goqu.I(CommentsLikes).Desc())
		} else if *opts.PageOpts.Order == store.OrderByDislikes {
			sd = sd.Order(goqu.I(CommentsDislikes).Desc())
		} else if *opts.PageOpts.Order == store.OrderByBoth {
			sd = sd.Order(goqu.I(CommentsScore).Desc())
		} else {
			return nil, fmt.Errorf("unexpected ordering directive %d", *opts.PageOpts.Order)
		}
	}

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]*store.Comment, 0)
	var id, articleID, commentsArticleID, userID, commentsUserID int
	var tm time.Time
	var text string
	var likes, dislikes int32
	var score int64
	var deleted bool
	var articleTitle string
	var articleUrl string
	var userName string
	for rows.Next() {
		err := rows.Scan(&id, &tm, &text, &likes, &dislikes, &score, &deleted, &commentsArticleID, &commentsUserID, &articleID, &articleTitle, &articleUrl, &userID, &userName)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &store.Comment{
			ID:       id,
			Article:  store.Article{ID: articleID, Title: articleTitle, Url: articleUrl},
			User:     store.User{ID: userID, UserName: userName},
			Time:     tm.Local(),
			Text:     text,
			Likes:    likes,
			Dislikes: dislikes,
			Deleted:  deleted,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		return nil, &store.NoQueryResultsError{}
	}

	return comments, nil
}

func (s *sqlStorage) AddArticles(ctx context.Context, articles ...*store.Article) error {
	ds := s.dialect.Insert(ArticlesTable).Cols(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		OnConflict(goqu.DoNothing())

	// We want to set the lastScrapedTime to nil so that the article will be scraped immediately
	for _, article := range articles {
		ds = ds.Vals(goqu.Vals{article.ID, article.Url, article.Title, article.DiscoveryTime, nil})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) GetArticles(ctx context.Context, ids ...int) ([]*store.Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(goqu.Ex{ArticlesID: ids})

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return hydrateArticles(rows)
}

func (s *sqlStorage) AddUsers(ctx context.Context, users ...*store.User) error {
	ds := s.dialect.Insert(UsersTable).Cols(UsersID, UsersName).OnConflict(goqu.DoNothing()) // TODO Something other than nothing
	for _, user := range users {
		ds = ds.Vals(goqu.Vals{user.ID, user.UserName})
	}

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func addPaging(sd *goqu.SelectDataset, pageOpts *store.PageQueryOptions) *goqu.SelectDataset {
	if pageOpts == nil {
		return sd
	}

	limit := maxPageSize
	if pageOpts.Limit != nil && *pageOpts.Limit < limit {
		limit = *pageOpts.Limit
	}
	sd = sd.Limit(limit)

	page := uint(0)
	if pageOpts.Page != nil {
		page = *pageOpts.Page
	}
	sd = sd.Offset(page * limit)
	return sd
}

func hydrateArticles(rows *sql.Rows) ([]*store.Article, error) {
	articles := make([]*store.Article, 0)
	var id int
	var url, title string
	var first, last sql.NullTime
	for rows.Next() {
		err := rows.Scan(&id, &url, &title, &first, &last)
		if err != nil {
			return nil, err
		}

		article := &store.Article{
			ID:            id,
			Url:           url,
			Title:         title,
			DiscoveryTime: first.Time.Local(),
		}

		if last.Valid {
			localTime := last.Time.Local()
			article.LastScrapeTime = localTime
		}

		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(articles) == 0 {
		return nil, &store.NoQueryResultsError{}
	}

	return articles, nil
}

func (s *sqlStorage) GetUnscrapedArticlesSince(ctx context.Context, scrapeThreshold time.Time) ([]*store.Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(
			goqu.Or(
				goqu.Ex{
					ArticlesLastScrapeTime: nil,
				},
				goqu.Ex{
					ArticlesDiscoveryTime: goqu.Op{"lte": scrapeThreshold.Truncate(time.Second)},
				},
			),
		)

	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles, err := hydrateArticles(rows)
	if err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *sqlStorage) GetRecentlyDiscoveredArticles(ctx context.Context, threshold time.Time) ([]*store.Article, error) {
	sd := s.dialect.
		Select(ArticlesID, ArticlesUrl, ArticlesTitle, ArticlesDiscoveryTime, ArticlesLastScrapeTime).
		From(ArticlesTable).
		Where(
			goqu.Ex{
				ArticlesLastScrapeTime: goqu.Op{"gte": threshold.Truncate(time.Second)},
			},
		)
	query, _, err := sd.ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles, err := hydrateArticles(rows)
	if err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *sqlStorage) SetArticleScrapedNow(ctx context.Context, articleIDs ...int) error {
	ds := s.dialect.Update(ArticlesTable).
		Where(goqu.Ex{ArticlesID: articleIDs}).
		Set(goqu.Record{ArticlesLastScrapeTime: time.Now().Truncate(time.Second)})

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) SetArticleScrapedAt(ctx context.Context, scrapedTime time.Time, articleIDs ...int) error {
	ds := s.dialect.Update(ArticlesTable).
		Where(goqu.Ex{ArticlesID: articleIDs}).
		Set(goqu.Record{ArticlesLastScrapeTime: scrapedTime.Truncate(time.Second)})

	query, _, err := ds.ToSQL()
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query)
	return err
}

func (s *sqlStorage) shutdown() error {
	return s.db.Close()
}
