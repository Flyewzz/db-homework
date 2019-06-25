package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/Flyewzz/db-homework/models"
	"github.com/gorilla/mux"
)

// /post/{id}/details GET (1)
func getPostFromDatabase(id int) (*models.Post, error) {
	post := &models.Post{}

	err := DB.pool.QueryRow(
		`SELECT id, author, message, forum, thread, created, "isEdited", parent
		FROM posts 
		WHERE id = $1`,
		id,
	).Scan(
		&post.Author,
		&post.Message,
		&post.Forum,
		&post.Thread,
		&post.Created,
		&post.IsEdited,
		&post.Parent,
	)

	if err == nil {
		return post, nil
	} else if err.Error() == noRowsInResult {
		return nil, PostIsNotFound
	} else {
		// ...
		return nil, err
	}
}

// /post/{id}/details GET (2, with related)
func GetPost(w http.ResponseWriter, r *http.Request) {
	// (id int, related []string) (*models.PostFull, error)
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	queryParams := r.URL.Query()
	relatedQuery := queryParams.Get("related")
	related := []string{}
	related = append(related, strings.Split(string(relatedQuery), ",")...)

	postFull := models.PostFull{}

	postFull.Post, err = getPostFromDatabase(id)
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	for _, model := range related {
		switch model {
		case "thread":
			postFull.Thread, err = func(param string) (*models.Thread, error) {
				var err error
				var thread models.Thread

				if slugIsNumber(param) {
					id, _ := strconv.Atoi(param)
					err = DB.pool.QueryRow(
						`SELECT id, title, author, forum, message, votes, slug, created
						FROM threads
						WHERE id = $1`,
						id,
					).Scan(
						&thread.Id,
						&thread.Title,
						&thread.Author,
						&thread.Forum,
						&thread.Message,
						&thread.Votes,
						&thread.Slug,
						&thread.Created,
					)
				} else {
					err = DB.pool.QueryRow(
						`SELECT id, title, author, forum, message, votes, slug, created
						FROM threads
						WHERE slug = $1`,
						param,
					).Scan(
						&thread.Id,
						&thread.Title,
						&thread.Author,
						&thread.Forum,
						&thread.Message,
						&thread.Votes,
						&thread.Slug,
						&thread.Created,
					)
				}

				if err != nil {
					return nil, ThreadIsNotFound
				}

				return &thread, nil
			}(strconv.Itoa(int(postFull.Post.Thread)))
		case "forum":
			postFull.Forum, err = func(slug string) (*models.Forum, error) {
				f := models.Forum{}

				err := DB.pool.QueryRow(
					`SELECT slug, title, "user", posts, threads
					FROM forums
					WHERE slug = $1`,
					slug,
				).Scan(
					&f.Slug,
					&f.Title,
					&f.User,
					&f.Posts,
					&f.Threads,
				)

				if err != nil {
					return nil, ForumIsNotFound
				}

				return &f, nil
			}(postFull.Post.Forum)
		case "user":
			postFull.Author, err = func(nickname string) (*models.User, error) {
				user := models.User{}

				err := DB.pool.QueryRow(
					`SELECT "nickname", "fullname", "email", "about"
					FROM users
					WHERE "nickname" = $1`,
					nickname,
				).Scan(
					&user.Nickname,
					&user.Fullname,
					&user.Email,
					&user.About,
				)

				if err != nil {
					return nil, UserIsNotFound
				}

				return &user, nil
			}(postFull.Post.Author)
		}

		// if err != nil {
		// 	sendResponse(w, 500, []byte(err.Error()))
		// }

		switch err {
		case nil:
			resp, _ := json.Marshal(postFull)
			sendResponse(w, 200, resp)
		case PostIsNotFound:
			sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post with id: %s"}`, string(id))))
		default:
			sendResponse(w, 500, []byte(err.Error()))
		}
	}
}

// /post/{id}/details UPDATE
func UpdatePost(w http.ResponseWriter, r *http.Request) {
	//  (postUpdate *models.PostUpdate, id int) (*models.Post, error)
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}
	postUpdate := &models.PostUpdate{}
	err = json.Unmarshal(body, postUpdate)

	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	post, err := getPostFromDatabase(id)
	if err != nil {
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post with id: %s"}`, string(id))))
	}

	// if len(postUpdate.Message) == 0 {
	// 	return post, nil
	// }

	err = DB.pool.QueryRow(`
	UPDATE posts SET message=COALESCE(NULLIF($2,''),message), "isEdited" = CASE 
	WHEN message=$2 THEN false
	WHEN $2='' THEN false
	ELSE true
	END
	WHERE id=$1
	RETURNING author::TEXT, created, forum::TEXT, id, "isEdited", message::TEXT, parent, thread, slug
	`,
		strconv.Itoa(id),
		&postUpdate.Message).Scan(
		&post.Author,
		&post.Created,
		&post.Forum,
		&post.IsEdited,
		&post.Thread,
		&post.Message,
		&post.Parent,
	)

	if err != nil {
		if err.Error() == noRowsInResult {
			sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post with id: %s"}`, string(id))))
			return
		}
	}

	switch err {
	case nil:
		message, _ := json.Marshal(post)
		sendResponse(w, 200, message)
	case PostIsNotFound:
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post with id: %s"}`, string(id))))
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}
