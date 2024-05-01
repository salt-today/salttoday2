-- +migrate Up

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

DROP VIEW CommentsControversy;
