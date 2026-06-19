-- +migrate Up

-- Viafoura-backed sites (e.g. lfpress.com) identify articles, comments, and users
-- by UUIDs rather than integers. Widen the ID columns to strings so both Village
-- Media integer IDs (stored as their decimal string) and Viafoura UUIDs fit.
-- The CommentControversy view reads Comments.ID, so it must be dropped and
-- recreated around the column type change.

DROP VIEW IF EXISTS CommentControversy;

ALTER TABLE Comments
    MODIFY ID VARCHAR(64) NOT NULL,
    MODIFY ArticleID VARCHAR(64) NOT NULL,
    MODIFY UserID VARCHAR(64) NOT NULL;

ALTER TABLE Articles
    MODIFY ID VARCHAR(64) NOT NULL;

ALTER TABLE Users
    MODIFY ID VARCHAR(64) NOT NULL;

CREATE VIEW CommentControversy AS
SELECT
  ID,
  LOG2(Likes + Dislikes + 1) * -(
    ((0.00001 + likes) / (0.00002 + Likes + Dislikes)) * LOG2(
      (0.00001 + likes) / (0.00002 + Likes + Dislikes) + 0.00001
    ) + (
      (
        1 - ((0.00001 + Likes) / (0.00002 + Likes + Dislikes))
      ) * LOG2(
        1 - ((0.00001 + Likes) / (0.00002 + Likes + Dislikes)) + 0.00001
      )
    )
  ) AS WeightedEntropy
FROM
  Comments;

-- +migrate Down

DROP VIEW IF EXISTS CommentControversy;

ALTER TABLE Comments
    MODIFY ID INT NOT NULL,
    MODIFY ArticleID INT NOT NULL,
    MODIFY UserID INT NOT NULL;

ALTER TABLE Articles
    MODIFY ID INT NOT NULL;

ALTER TABLE Users
    MODIFY ID INT NOT NULL;
