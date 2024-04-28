-- +migrate Up

FLUSH PRIVILEGES;
CREATE USER IF NOT EXISTS 'salt'@'%' IDENTIFIED BY 'salt';
GRANT ALL PRIVILEGES ON *.* TO 'salt'@'%';
FLUSH PRIVILEGES;

-- DROP DATABASE salt;
CREATE DATABASE IF NOT EXISTS salt;
USE salt;
CREATE TABLE IF NOT EXISTS Comments (
    ID INT NOT NULL,
    ArticleID INT NOT NULL,
    UserID INT NOT NULL,
    Time DATETIME NOT NULL,
    Text MEDIUMTEXT NOT NULL,
    Likes INT NOT NULL DEFAULT(0),
    Dislikes INT NOT NULL DEFAULT(0),
    Deleted BOOLEAN NOT NULL DEFAULT(FALSE),
    PRIMARY KEY (ID),
    INDEX `user` (UserID)
);

CREATE TABLE IF NOT EXISTS Articles (
    ID INT NOT NULL,
    Url VARCHAR(2048) NOT NULL,
    Title VARCHAR(2048) NOT NULL,
    DiscoveryTime DATETIME NOT NULL,
    LastScrapeTime DATETIME,
    PRIMARY KEY (ID),
    INDEX scrape_time (LastScrapeTime)
);

CREATE TABLE IF NOT EXISTS Users (
    ID INT NOT NULL,
    Name VARCHAR(255) NOT NULL,
    PRIMARY KEY (ID)
);

-- +migrate Down

DROP TABLE Comments;
DROP TABLE Articles;
DROP TABLE Users;
