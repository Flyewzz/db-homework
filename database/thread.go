package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Flyewzz/db-homework/models"
	"github.com/go-openapi/swag"
	"github.com/gorilla/mux"
)

// Slug can be string or integer
func slugIsNumber(slug string) bool {
	if _, err := strconv.Atoi(slug); err != nil {
		return false
	}
	return true
}

func checkPost(p *models.Post, t *models.Thread) error {
	if func(nickname string) bool {
		var user models.User
		err := DB.pool.QueryRow(
			`SELECT "nickname", "fullname", "email", "about"
			FROM users
			WHERE "nickname" = $1`,
			nickname,
		).Scan(
			&user.Nickname,
			&user.Fullname,
			&user.About,
			&user.Email,
		)

		if err != nil && err.Error() == noRowsInResult {
			return true
		}
		return false
	}(p.Author) {
		return UserIsNotFound
	}
	if func(parent int64, threadID int32) bool {
		var t int64
		err := DB.pool.QueryRow(`
		SELECT id
		FROM posts
		WHERE id = $1 AND thread IN (SELECT id FROM threads WHERE thread <> $2)`,
			parent,
			threadID).Scan(&t)

		if err != nil && err.Error() == noRowsInResult {
			return false
		}
		return true
	}(p.Parent, t.Id) || func(parent int64) bool {
		if parent == 0 {
			return false
		}

		var t int64
		err := DB.pool.QueryRow(`SELECT id FROM posts WHERE id = $1`, parent).Scan(&t)

		if err != nil {
			return true
		}
		return false
	}(p.Parent) {
		return PostParentIsNotFound
	}
	return nil
}

func GetThreadFromDatabase(slug string) (*models.Thread, error) {
	thread := &models.Thread{}

	if slugIsNumber(slug) {
		id, _ := strconv.Atoi(slug)
		if err := DB.pool.QueryRow(
			`SELECT author, created, forum, id, message, slug, title, votes
			FROM Threads
			WHERE id = $1::INTEGER;`, id).Scan(
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.Id,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		); err != nil {
			return nil, errors.New("Thread is not found")
		}
		return thread, nil
	} else {
		// slug is string
		if err := DB.pool.QueryRow(
			`SELECT author, created, forum, id, message, slug, title, votes
			FROM Threads
			WHERE slug = $1::TEXT;`, slug).Scan(
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.Id,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		); err != nil {
			return nil, errors.New("Thread is not found")
		}
		return thread, nil
	}
}

// /thread/{slug_or_id}/details Получение информации о ветке обсуждения
func GetThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug_or_id"]

	thread := &models.Thread{}
	var err error

	if slugIsNumber(slug) {
		id, _ := strconv.Atoi(slug)
		if err = DB.pool.QueryRow(
			`SELECT author, created, forum, id, message, slug, title, votes
			FROM Threads
			WHERE id = $1::INTEGER;`, id).Scan(
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.Id,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		); err != nil {
			sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
			return
		}
		message, _ := json.Marshal(thread)
		sendResponse(w, 200, message)
	} else {
		// slug is string
		if err = DB.pool.QueryRow(
			`SELECT author, created, forum, id, message, slug, title, votes
			FROM Threads
			WHERE slug = $1::TEXT;`, slug).Scan(
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.Id,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		); err != nil {
			sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
			return
		}
		// return thread, nil
	}

	switch err {
	case nil:
		message, _ := json.Marshal(thread)
		sendResponse(w, 200, message)
	case ThreadIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}

func CreatePostsOnThreadDatabase(slug string, posts []*models.Post) ([]*models.Post, error) {
	thread, err := GetThreadFromDatabase(slug)
	if err != nil {
		return nil, err
	}
	dateTimeTemplate := "2000-01-01 12:00:00"
	dateTimeCreated := time.Now().Format(dateTimeTemplate)
	query := strings.Builder{}
	query.WriteString("INSERT INTO posts (author, created, message, thread, parent, forum, path) VALUES ")
	queryBody := "('%s', '%s', '%s', %d, %d, '%s', (SELECT  FROM posts WHERE id = %d) || (SELECT last_value FROM posts_id_seq)),"
	for i, post := range *posts {
		err = checkPost(post, thread)
		if err != nil {
			return nil, err
		}

		temp := fmt.Sprintf(queryBody, post.Author, created, post.Message, thread.Id, post.Parent, thread.Forum, post.Parent)
		if i == postsNumber-1 {
			temp = temp[:len(temp)-1]
		}
		query.WriteString(temp)
	}
	query.WriteString("RETURNING author, created, forum, id, message, parent, thread")

	tx, txErr := DB.pool.Begin()
	if txErr != nil {
		return nil, txErr
	}
	defer tx.Rollback()

	rows, err := tx.Query(query.String())
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	insertPosts := models.Posts{}
	for rows.Next() {
		post := models.Post{}
		rows.Scan(
			&post.Author,
			&post.Created,
			&post.Forum,
			&post.ID,
			&post.Message,
			&post.Parent,
			&post.Thread,
		)
		insertPosts = append(insertPosts, &post)
	}
	err = rows.Err()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// по хорошему это впихнуть в хранимые процедуры, но нормальные ребята предпочитают костылить
	tx.Exec(`UPDATE forums SET posts = posts + $1 WHERE slug = $2`, len(insertPosts), thread.Forum)
	for _, p := range insertPosts {
		tx.Exec(`INSERT INTO forum_users VALUES ($1, $2) ON CONFLICT DO NOTHING`, p.Author, p.Forum)
	}

	tx.Commit()

	return &insertPosts, nil
}

// /thread/{slug_or_id}/create Создание новых постов
func CreatePost(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("/thread/{slug_or_id}/create")
	params := mux.Vars(r)
	param := params["slug_or_id"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		makeResponse(w, 500, []byte(err.Error()))
		return
	}
	posts := &models.Posts{}
	err = json.Unmarshal(body, &posts)
	if err != nil {
		makeResponse(w, 500, []byte(err.Error()))
		return
	}

	result, err := database.CreateThreadDB(posts, param)

	resp, _ := swag.WriteJSON(result)

	switch err {
	case nil:
		makeResponse(w, 201, resp)
	case database.ThreadNotFound:
		makeResponse(w, 404, []byte(makeErrorThreadID(param)))
	case database.UserNotFound:
		makeResponse(w, 404, []byte(makeErrorPostAuthor(param)))
	case database.PostParentNotFound:
		makeResponse(w, 409, []byte(makeErrorThreadConflict()))
	default:
		makeResponse(w, 500, []byte(err.Error()))
	}
}

// /thread/{slug_or_id}/posts Сообщения данной ветви обсуждения
func GetThreadPosts(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	param := params["slug_or_id"]
	queryParams := r.URL.Query()
	var limit, since, sort, desc string
	if limit = queryParams.Get("limit"); limit == "" {
		limit = "1"
	}
	since = queryParams.Get("since")

	if sort = queryParams.Get("sort"); sort == "" {
		sort = "flat"
	}
	if desc = queryParams.Get("desc"); desc == "" {
		desc = "false"
	}
	result, err := database.GetThreadPostsDB(param, limit, since, sort, desc)

	resp, _ := swag.WriteJSON(result)
	switch err {
	case nil:
		makeResponse(w, 200, resp)
	case database.ForumNotFound:
		makeResponse(w, 404, []byte(makeErrorThread(param)))
	default:
		makeResponse(w, 500, []byte(err.Error()))
	}
}

// /thread/{slug_or_id}/vote Проголосовать за ветвь обсуждения
func MakeThreadVote(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	param := params["slug_or_id"]
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		makeResponse(w, 500, []byte(err.Error()))
		return
	}
	vote := &models.Vote{}
	err = vote.UnmarshalJSON(body)

	result, err := database.MakeThreadVoteDB(vote, param)

	switch err {
	case nil:
		resp, _ := result.MarshalJSON()
		makeResponse(w, 200, resp)
	case database.ForumNotFound:
		makeResponse(w, 404, []byte(makeErrorThread(param)))
	case database.UserNotFound:
		makeResponse(w, 404, []byte(makeErrorUser(param)))
	default:
		makeResponse(w, 500, []byte(err.Error()))
	}
}
