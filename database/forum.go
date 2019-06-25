package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Flyewzz/db-homework/models"
	"github.com/go-openapi/swag"
	"github.com/gorilla/mux"
)

// /forum/{slug}/details GET
func GetForum(w http.ResponseWriter, r *http.Request) {
	// (slug string) (*models.Forum, error)
	params := mux.Vars(r)
	slug := params["slug"]
	forum := models.Forum{} // Create an empty forum

	if err := DB.pool.QueryRow(
		`SELECT slug, title, "user", posts, threads
			FROM Forums
			WHERE slug = $1;`,
		slug,
	).Scan(
		&forum.Slug,
		&forum.Title,
		&forum.User,
		&forum.Posts,
		&forum.Threads,
	); err != nil {
		forum = nil
		err = ForumIsNotFound
	}
	switch err {
	case nil:
		message, _ := json.Marshal(forum)
		SendResponse(w, 200, message)
	case database.ForumNotFound:
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find forum with slug: %s"}`, slug)))
	default:
		SendResponse(w, 500, []byte(err.Error()))
	}
}

// /forum/create POST
func CreateForum(w http.ResponseWriter, r *http.Request) {
	// (forum *models.Forum) (*models.Forum, error)
	// Check for error
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		SendResponse(w, 500, []byte(err.Error()))
		return
	}
	forum := &models.Forum{}
	err = json.Unmarshal(body, forum)

	if err != nil {
		SendResponse(w, 500, []byte(err.Error()))
		return
	}

	err := DB.pool.QueryRow(
		`INSERT INTO Forums (slug, title, "user")
		VALUES ($1, $2, (
			SELECT nickname FROM Users WHERE nickname = $3
		))
		RETURNING "user";
		`,
		&forum.Slug,
		&forum.Title,
		&forum.User,
	).Scan(&forum.User)

	switch ErrorCode(err) {
	case pgxStatusOk:
		message, _ := json.Marshal(forum)
		SendResponse(w, 201, message)
	case pgxStatusErrorUnique:
		message, _ := json.Marshal(forum)
		SendResponse(w, 409, message)
	case pgxStatusErrorNotNull:
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find user by nickname: %s"}`, forum.User)))
	default:
		SendResponse(w, 500, []byte(err.Error()))
	}
}

// /forum/{slug}/users GET
func GetForumUsers(w http.ResponseWriter, r *http.Request) {
	// (slug, limit, since, desc string) (*[]*models.User, error)
	params := mux.Vars(r)
	slug := params["slug"]
	queryParams := r.URL.Query()
	var limit, since, desc string
	if limit = queryParams.Get("limit"); limit == "" {
		limit = "1"
	}
	since = queryParams.Get("since")
	// if since = queryParams.Get("since"); since == "" {
	// 	since = "";
	// }
	if desc = queryParams.Get("desc"); desc == "" {
		desc = "false"
	}

	query := `SELECT about, email, fullname, nickname
			  FROM Users
			  WHERE nickname IN (
				  SELECT "user" FROM ForumsUsers
				  WHERE forum = $1
			  ) AND LOWER(nickname) > LOWER($3::TEXT)
			  LIMIT $2::INTEGER ORDER BY `
	if desc == "false" {
		query += "ASC"
	} else {
		query += "DESC"
	}
	results, err := DB.pool.Query(query, slug, limit, since)
	if err != nil {
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find user by nickname: %s"}`, slug)))
	}
	defer results.Close()
	var users []*models.User
	for results.Next() {
		user := &models.User{}
		results.Scan(
			&user.About,
			&user.Email,
			&user.Fullname,
			&user.Nickname,
		)
		// Add a new user
		users = append(users, user)
	}

	switch err {
	case nil:
		message, _ := swag.WriteJSON(users) // можно через easyjson, но мне лень было
		SendResponse(w, 200, message)
	case ForumIsNotFound:
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find user by nickname: %s"}`, slug)))
	default:
		SendResponse(w, 500, []byte(err.Error()))
	}
}

// /forum/{slug}/create POST
func CreateForumThread(w http.ResponseWriter, r *http.Request) {
	// (slug string, thread *models.Thread) (*models.Thread, error)
	params := mux.Vars(r)
	slug := params["slug"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		SendResponse(w, 500, []byte(err.Error()))
		return
	}
	thread := &models.Thread{}
	err = json.Unmarshal(body, thread)
	thread.Forum = slug // иначе не знаю как

	if err != nil {
		SendResponse(w, 500, []byte(err.Error()))
		return
	}

	// if t, _ := GetThreadFromDatabase(thread.Slug); t != nil {
	// 	return nil, errors.New("Thread already exists")
	// }

	// Thread not exists
	err := DB.pool.QueryRow(
		`INSERT INTO threads (author, created, message, title, slug, forum)
		 VALUES ($1, $2, $3, $4, $5, (SELECT slug FROM forums where slug = $6))
		 RETURNING author, created, forum, id, message, title;
		`,
		&thread.Author,
		&thread.Created,
		&thread.Message,
		&thread.Title,
		&thread.Slug,
		&thread.Forum,
	).Scan(
		&thread.Author,
		&thread.Created,
		&thread.Forum,
		&thread.Id,
		&thread.Message,
		&thread.Title,
	)
	switch ErrorCode(err) {
	case pgxStatusOk:
		return thread, nil
	case pgxStatusErrorWithForeignKey:
		fallthrough
	case pgxStatusErrorNotNull:
		thread = nil
		err = errors.New("Author/Forum is not found")
	default:
		thread = nil
	}

	switch err {
	case nil:
		message, _ := json.MarshalJSON(thread)
		SendResponse(w, 201, message)
	case ForumOrAuthorIsNotFound:
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find user by nickname: %s"}`, slug)))
	case ThreadIsExist:
		message, _ := json.MarshalJSON(thread)
		SendResponse(w, 409, message)
	default:
		SendResponse(w, 500, []byte(err.Error()))
	}

}

// /forum/{slug}/threads GET
func GetForumThreads(w http.ResponseWriter, r *http.Request) {
	// // (slug, limit, since, desc string) (*[]*models.Thread, error)
	params := mux.Vars(r)
	slug := params["slug"]
	queryParams := r.URL.Query()
	var limit, since, desc string
	if limit = queryParams.Get("limit"); limit == "" {
		limit = "1"
	}
	since = queryParams.Get("since")
	if desc = queryParams.Get("desc"); desc == "" {
		desc = "false"
	}

	query := `SELECT author, created, forum, id, message, slug, title, votes
			  FROM Threads
			  WHERE forum = $1 AND created >= $2::TIMESTAMP
			  LIMIT $3::INTEGER ORDER BY `
	if desc == "false" {
		query += "ASC"
	} else {
		query += "DESC"
	}
	results, err := DB.pool.Query(query, slug, limit, since)
	var threads []*models.Thread
	if err != nil {
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find forum with slug: %s"}`, slug)))
		return
	}
	defer results.Close()
	for results.Next() {
		thread := &models.Thread{}
		results.Scan(
			&thread.Author,
			&thread.Created,
			&thread.Forum,
			&thread.Id,
			&thread.Message,
			&thread.Slug,
			&thread.Title,
			&thread.Votes,
		)
		// Add a new user
		threads = append(threads, thread)
	}

	switch err {
	case nil:
		message, _ := swag.WriteJSON(threads)
		SendResponse(w, 200, message)
	case ForumIsNotFound:
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find forum with slug: %s"}`, slug)))
	default:
		SendResponse(w, 500, []byte(err.Error()))
	}
}
