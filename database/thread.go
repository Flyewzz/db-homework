package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Flyewzz/db-homework/models"
	"github.com/go-openapi/swag"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
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
	}(p.Parent, t.ID) || func(parent int64) bool {
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
	thread := models.Thread{}

	if slugIsNumber(slug) {
		id, _ := strconv.Atoi(slug)
		if err := DB.pool.QueryRow(
			`SELECT author, created, forum, id, message, slug, title, votes
			FROM Threads
			WHERE id = $1::INTEGER;`, id).Scan(
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.ID,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		); err != nil {
			return nil, ThreadIsNotFound
		}
		return &thread, nil
	} else {
		// slug is string
		if err := DB.pool.QueryRow(
			`SELECT author, created, forum, id, message, slug, title, votes
			FROM Threads
			WHERE slug = $1::TEXT;`, slug).Scan(
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.ID,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		); err != nil {
			return nil, ThreadIsNotFound
		}
		return &thread, nil
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
			&thread.ID,
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
			&thread.ID,
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

// /thread/{slug_or_id}/details POST
func UpdateThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug_or_id"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}
	threadUpdate := &models.ThreadUpdate{}
	err = json.Unmarshal(body, threadUpdate)

	//err = forum.Validate()
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	threadFound, err := GetThreadFromDatabase(slug)
	if err != nil {
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
	}

	updatedThread := models.Thread{}

	err = DB.pool.QueryRow(
		`UPDATE threads
		SET title = coalesce(nullif($2, ''), title),
			message = coalesce(nullif($3, ''), message)
		WHERE slug = $1
		RETURNING id, title, author, forum, message, votes, slug, created`,
		&threadFound.Slug,
		&threadUpdate.Title,
		&threadUpdate.Message,
	).Scan(
		&updatedThread.ID,
		&updatedThread.Title,
		&updatedThread.Author,
		&updatedThread.Forum,
		&updatedThread.Message,
		&updatedThread.Votes,
		&updatedThread.Slug,
		&updatedThread.Created,
	)

	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	switch err {
	case nil:
		message, _ := json.Marshal(updatedThread)
		sendResponse(w, 200, message)
	case PostIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}

func CreatePostsOnThreadDatabase(slug string, posts *[]*models.Post) (*[]*models.Post, error) {
	thread, err := func(param string) (*models.Thread, error) {
		thread := models.Thread{}
		if slugIsNumber(slug) {
			id, _ := strconv.Atoi(slug)
			if err := DB.pool.QueryRow(
				`SELECT author, created, forum, id, message, slug, title, votes
			FROM Threads
			WHERE id = $1::INTEGER;`, id).Scan(
				&thread.Author,
				&thread.Created,
				&thread.Forum,
				&thread.ID,
				&thread.Message,
				&thread.Slug,
				&thread.Title,
				&thread.Votes,
			); err != nil {
				return nil, ThreadIsNotFound
			}
			return &thread, nil
		} else {
			// slug is string
			if err := DB.pool.QueryRow(
				`SELECT author, created, forum, id, message, slug, title, votes
					FROM Threads
					WHERE slug = $1::TEXT;`, slug).Scan(
				&thread.Author,
				&thread.Created,
				&thread.Forum,
				&thread.ID,
				&thread.Message,
				&thread.Slug,
				&thread.Title,
				&thread.Votes,
			); err != nil {
				return nil, ThreadIsNotFound
			}
			return &thread, nil
		}
	}(slug)
	if err != nil {
		return nil, err
	}

	postsNumber := len(*posts)
	if postsNumber == 0 {
		return posts, nil
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

		temp := fmt.Sprintf(queryBody, post.Author, dateTimeCreated, post.Message, thread.ID, post.Parent, thread.Forum, post.Parent)
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
	insertPosts := []*models.Post{}
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

	tx.Exec(`UPDATE forums SET posts = posts + $1 WHERE slug = $2`, len(insertPosts), thread.Forum)
	for _, p := range insertPosts {
		tx.Exec(`INSERT INTO forum_users VALUES ($1, $2) ON CONFLICT DO NOTHING`, p.Author, p.Forum)
	}

	tx.Commit()

	return &insertPosts, nil
}

// /thread/{slug_or_id}/create POST
func CreatePost(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("/thread/{slug_or_id}/create")
	params := mux.Vars(r)
	slug := params["slug_or_id"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}
	posts := &[]*models.Post{}
	err = json.Unmarshal(body, &posts)
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	conclusion, err := CreatePostsOnThreadDatabase(slug, posts)
	message, _ := swag.WriteJSON(conclusion)

	switch err {
	case nil:
		sendResponse(w, 201, message)
	case PostParentIsNotFound:
		sendResponse(w, 409, []byte(`{"message": "Parent post was created in another thread"}`))
	case ThreadIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
	case UserIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post author by nickname: %s"}`, slug)))
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}

func getPosts(slug, limit, since, sort, desc string) (*[]*models.Post, error) {
	thread, err := GetThreadFromDatabase(slug)
	if err != nil {
		return nil, ForumIsNotFound
	}

	var rows *pgx.Rows

	var queryPostsWithSience = map[string]map[string]string{
		"true": map[string]string{
			"tree": `SELECT id, author, parent, message, forum, thread, created
						FROM posts
						WHERE thread = $1 AND (path < (SELECT path FROM posts WHERE id = $2::TEXT::INTEGER))
						ORDER BY path DESC
						LIMIT $3::TEXT::INTEGER`,
			"parent_tree": `SELECT id, author, parent, message, forum, thread, created
							FROM posts p
							WHERE p.thread = $1 and p.path[1] IN (
								SELECT p2.path[1]
								FROM posts p2
								WHERE p2.thread = $1 AND p2.parent = 0 and p2.path[1] < (SELECT p3.path[1] from posts p3 where p3.ID = $2)
								ORDER BY p2.path DESC
								LIMIT $3
							)
							ORDER BY p.path[1] DESC, p.path[2:]`,
			"flat": `SELECT id, author, parent, message, forum, thread, created
						FROM posts
						WHERE thread = $1 AND id < $2::TEXT::INTEGER
						ORDER BY id DESC
						LIMIT $3::TEXT::INTEGER`,
		},
		"false": map[string]string{
			"tree": `SELECT id, author, parent, message, forum, thread, created
				FROM posts
				WHERE thread = $1 AND (path > (SELECT path FROM posts WHERE id = $2::TEXT::INTEGER))
				ORDER BY path
				LIMIT $3::TEXT::INTEGER`,
			"parent_tree": `SELECT id, author, parent, message, forum, thread, created
							FROM posts p
							WHERE p.thread = $1 and p.path[1] IN (
								SELECT p2.path[1]
								FROM posts p2
								WHERE p2.thread = $1 AND p2.parent = 0 and p2.path[1] > (SELECT p3.path[1] from posts p3 where p3.ID = $2::TEXT::INTEGER)
								ORDER BY p2.path
								LIMIT $3::TEXT::INTEGER
							)
							ORDER BY p.path`,
			"flat": `SELECT id, author, parent, message, forum, thread, created
							FROM posts
							WHERE thread = $1 AND id > $2::TEXT::INTEGER
							ORDER BY id
							LIMIT $3::TEXT::INTEGER`,
		},
	}

	var queryPostsWithNoSience = map[string]map[string]string{
		"true": map[string]string{
			"tree": `SELECT id, author, parent, message, forum, thread, created
						FROM posts
						WHERE thread = $1 
						ORDER BY path DESC
						LIMIT $2::TEXT::INTEGER`,
			"parent_tree": `SELECT id, author, parent, message, forum, thread, created
							FROM posts
							WHERE thread = $1 AND path[1] IN (
								SELECT path[1]
								FROM posts
								WHERE thread = $1
								GROUP BY path[1]
								ORDER BY path[1] DESC
								LIMIT $2::TEXT::INTEGER
							)
							ORDER BY path[1] DESC, path`,
			"flat": `SELECT id, author, parent, message, forum, thread, created
						FROM posts
						WHERE thread = $1
						ORDER BY id DESC
						LIMIT $2::TEXT::INTEGER`,
		},
		"false": map[string]string{
			"tree": `SELECT id, author, parent, message, forum, thread, created
						FROM posts
						WHERE thread = $1 
						ORDER BY path
						LIMIT $2::TEXT::INTEGER`,
			"parent_tree": `SELECT id, author, parent, message, forum, thread, created
							FROM posts
							WHERE thread = $1 AND path[1] IN (
								SELECT path[1] 
								FROM posts 
								WHERE thread = $1 
								GROUP BY path[1]
								ORDER BY path[1]
								LIMIT $2::TEXT::INTEGER
							)
							ORDER BY path`,
			"flat": `SELECT id, author, parent, message, forum, thread, created
						FROM posts
						WHERE thread = $1 
						ORDER BY id
						LIMIT $2::TEXT::INTEGER`,
		},
	}
	if since != "" {
		query := queryPostsWithSience[desc][sort]
		rows, err = DB.pool.Query(query, thread.ID, since, limit)
	} else {
		query := queryPostsWithNoSience[desc][sort]
		rows, err = DB.pool.Query(query, thread.ID, limit)
	}
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	posts := []*models.Post{}
	for rows.Next() {
		post := models.Post{}

		err = rows.Scan(
			&post.ID,
			&post.Author,
			&post.Parent,
			&post.Message,
			&post.Forum,
			&post.Thread,
			&post.Created,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return &posts, nil
}

// /thread/{slug_or_id}/posts Сообщения данной ветви обсуждения
func GetThreadPosts(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug_or_id"]
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
	res, err := getPosts(slug, limit, since, sort, desc)

	message, _ := swag.WriteJSON(res)
	switch err {
	case nil:
		sendResponse(w, 200, message)
	case ForumIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}

// /thread/{slug_or_id}/vote POST
func MakeThreadVote(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug_or_id"]
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}
	vote := &models.Vote{}
	err = json.Unmarshal(body, vote)
	res, err := func(vote *models.Vote, param string) (*models.Thread, error) {
		var err error

		tx, txErr := DB.pool.Begin()
		if txErr != nil {
			return nil, txErr
		}
		defer tx.Rollback()

		var thread models.Thread
		if slugIsNumber(param) {
			id, _ := strconv.Atoi(param)
			err = tx.QueryRow(`SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE id = $1`, id).Scan(
				&thread.ID,
				&thread.Author,
				&thread.Created,
				&thread.Forum,
				&thread.Message,
				&thread.Slug,
				&thread.Title,
				&thread.Votes,
			)
		} else {
			err = tx.QueryRow(`SELECT id, author, created, forum, message, slug, title, votes FROM threads WHERE slug = $1`, param).Scan(
				&thread.ID,
				&thread.Author,
				&thread.Created,
				&thread.Forum,
				&thread.Message,
				&thread.Slug,
				&thread.Title,
				&thread.Votes,
			)
		}
		if err != nil {
			return nil, ForumIsNotFound
		}

		var nick string
		err = tx.QueryRow(`SELECT nickname FROM users WHERE nickname = $1`, vote.Nickname).Scan(&nick)
		if err != nil {
			return nil, UserIsNotFound
		}

		rows, err := tx.Exec(`UPDATE votes SET voice = $1 WHERE thread = $2 AND nickname = $3;`, vote.Voice, thread.ID, vote.Nickname)
		if rows.RowsAffected() == 0 {
			_, err := tx.Exec(`INSERT INTO votes (nickname, thread, voice) VALUES ($1, $2, $3);`, vote.Nickname, thread.ID, vote.Voice)
			if err != nil {
				return nil, UserIsNotFound
			}
		}
		// если возник вопрос - в какой мемент делаем +1 к voice -> смотри init.sql

		err = tx.QueryRow(`SELECT votes FROM threads WHERE id = $1`, thread.ID).Scan(&thread.Votes)
		if err != nil {
			return nil, err
		}

		tx.Commit()

		return &thread, nil
	}(vote, slug)

	switch err {
	case nil:
		message, _ := json.Marshal(res)
		sendResponse(w, 200, message)
	case ForumIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find thread by slug: %s"}`, slug)))
	case UserIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find user by nickname: %s"}`, slug)))
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}
