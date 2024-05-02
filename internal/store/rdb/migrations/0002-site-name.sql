-- +migrate Up

ALTER TABLE Articles ADD COLUMN SiteName VARCHAR(30);

-- +migrate Down

