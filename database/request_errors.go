package database

import "errors"

var (
	ForumIsExist             = errors.New("Forum was created before")
	ForumIsNotFound          = errors.New("Forum is not found")
	AuthorOrForumAreNotFound = errors.New("Forum/Author not found")
	UserIsNotFound           = errors.New("User is not found")
	UserIsAlreadyExist       = errors.New("User was created before")
	UserUpdateErr            = errors.New("User is not updated")
	ThreadIsExist            = errors.New("Thread was created before")
	ThreadIsNotFound         = errors.New("Thread is not found")
	PostParentIsNotFound     = errors.New("Parents for this thread are not found")
	PostIsNotFound           = errors.New("Post is not found")
)
