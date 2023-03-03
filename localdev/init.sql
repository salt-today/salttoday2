CREATE DATABASE IF NOT EXISTS salt;
USE salt;
CREATE TABLE IF NOT EXISTS Comments (
    User VARCHAR(255) NOT NULL,
    Time DATETIME NOT NULL,
    Text MEDIUMTEXT NOT NULL,
    Likes INT NOT NULL DEFAULT(0),
    Dislikes INT NOT NULL DEFAULT(0),
    INDEX `user` (User)
);

CREATE TABLE IF NOT EXISTS Article (
    #can URL be my primary key? Should it be some generated ID.
    Url VARCHAR(2048) NOT NULL,
    Title VARCHAR(2048) NOT NULL,
    PRIMARY KEY (Url)
);

Create TABLE IF NOT EXISTS Users (
    #hmm what else to store here? Do I need it?
    User VARCHAR(255) NOT NULL,
    PRIMARY KEY (User)
);

FLUSH PRIVILEGES;
CREATE USER 'salt'@'%' IDENTIFIED BY 'salt';
GRANT ALL PRIVILEGES ON *.* TO 'salt'@'%';
FLUSH PRIVILEGES;
