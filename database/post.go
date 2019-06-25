package database

import (
	"strconv"

	"github.com/Flyewzz/db-homework/models"
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
func GetPost(id int, related []string) (*models.PostFull, error) {
	// (id int, related []string) (*models.PostFull, error)
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		makeResponse(w, 500, []byte(err.Error()))
		return
	}
	
	queryParams := r.URL.Query()
	relatedQuery := queryParams.Get("related")
	related := []string{}
	related = append(related, strings.Split(string(relatedQuery), ",")...)

	postFull := models.PostFull{}
	var err error
	postFull.Post, err = getPostFromDatabase(id)
	if err != nil {
		return nil, err
	}

	for _, model := range related {
		switch model {
		case "thread":
			postFull.Thread, err = GetThreadFromDatabase(strconv.Itoa(int(postFull.Post.Thread)))
		case "forum":
			postFull.Forum, err = GetForumDatabase(postFull.Post.Forum)
		case "user":
			// postFull.Author, err = GetUserDB(postFull.Post.Author)
		}

		if err != nil {
			return nil, err
		}

		switch err {
		case nil:
			resp, _ := json.Marshal(postFull)
			makeResponse(w, 200, resp)
		case database.PostNotFound:
			makeResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post with id: %s"}`, string(id)))))
		default:		
			makeResponse(w, 500, []byte(err.Error()))
		}
	}
}

// /post/{id}/details UPDATE
func UpdatePostDataBase(w http.ResponseWriter, *r http.Request) {
	//  (postUpdate *models.PostUpdate, id int) (*models.Post, error)
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
    if err != nil {
		SendResponse(w, 500, []byte(err.Error()))
		return
    }

	body, err := ioutil.ReadAll(r.Body)	
	defer r.Body.Close()
	if err != nil {
		SendResponse(w, 500, []byte(err.Error()))
		return
	}	
	postUpdate := &models.PostUpdate{}
	err = json.Unmarshal(body, postUpdate)

	if err != nil {
		SendResponse(w, 500, []byte(err.Error()))
		return
	}

	post, err := GetPostFromDatabase(id)
	if err != nil {
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post with id: %s"}`, string(id))))
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
		post = nil
		if err.Error() == noRowsInResult {
			err = PostIsNotFound
		}
	}

	switch err {
	case nil:
		resp, _ := json.Marshal(result)
		SendResponse(w, 200, resp)
	case PostNotFound:
		SendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find post with id: %s"}`, string(id))))
	default:		
		SendResponse(w, 500, []byte(err.Error()))
	}
}
