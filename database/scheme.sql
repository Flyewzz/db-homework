DROP TABLE IF EXISTS users, forums, threads, posts, votes, forum_users, posts_tree CASCADE;

-- CREATE EXTENSION IF NOT EXISTS citext;

-- USER --
CREATE TABLE users (
  nickname    CITEXT PRIMARY KEY,
  about       TEXT,
  email       CITEXT UNIQUE NOT NULL,
  fullname    CITEXT        NOT NULL
);

-- FORUM --
CREATE TABLE forums (
  slug        CITEXT  PRIMARY KEY,
  posts       INT NOT NULL DEFAULT 0,
  threads     INT NOT NULL DEFAULT 0,
  title       TEXT    NOT NULL,
  user_id     CITEXT  NOT NULL REFERENCES users
);

-- THREAD --
CREATE TABLE threads (
  id         SERIAL PRIMARY KEY,
  slug       CITEXT ,
  title      TEXT   NOT NULL,
  message    TEXT   NOT NULL,
  votes      INT    DEFAULT 0,
  created_at TIMESTAMPTZ(3) DEFAULT now(),
  forum_id   CITEXT NOT NULL REFERENCES forums,
  author_id  CITEXT NOT NULL REFERENCES users,
  UNIQUE (forum_id, slug)
);

-- POST --
CREATE TABLE posts (
  id                SERIAL PRIMARY KEY,
  message           TEXT    NOT NULL,
  is_edited         BOOLEAN NOT NULL DEFAULT FALSE,
  created_at        TIMESTAMPTZ DEFAULT now(),
--   root_id           INT     NOT NULL DEFAULT 0,
  parent_id         INT              DEFAULT 0,
--   path INT [],
  author_id         CITEXT  NOT NULL REFERENCES users,
  forum_id          CITEXT  NOT NULL REFERENCES forums,
  thread_id         INT     NOT NULL
);

-- VOTE --
CREATE TABLE votes (
  voice       INT      NOT NULL,
  nickname    CITEXT   NOT NULL REFERENCES users,
  thread_id   INT      NOT NULL REFERENCES threads
);