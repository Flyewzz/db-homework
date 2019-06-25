package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Flyewzz/db-homework/models"
	"github.com/go-openapi/swag"
	"github.com/gorilla/mux"
)

// /user/{nickname}/create POST;
func CreateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}
	user := &models.User{}
	err = json.Unmarshal(body, &user)
	user.Nickname = nickname

	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}

	results, err := DB.pool.Exec(
		`INSERT
		INTO users ("nickname", "fullname", "email", "about")
		VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING;`,
		&user.Nickname,
		&user.Fullname,
		&user.Email,
		&user.About,
	)

	var users models.Users
	if results.RowsAffected() == 0 {
		users = models.Users{}
		queryRows, err := DB.pool.Query(
			`SELECT "nickname", "fullname", "email", "about"
			 FROM users
			 WHERE "nickname" = $1 OR "email" = $2
			 `,
			&user.Nickname,
			&user.Email,
		)
		defer queryRows.Close()

		if err != nil {
			sendResponse(w, 500, []byte(err.Error()))
		}

		for queryRows.Next() {
			user := models.User{}
			queryRows.Scan(&user.Nickname, &user.Fullname, &user.Email, &user.About)
			users = append(users, &user)
		}
		message, _ := swag.WriteJSON(users)
		sendResponse(w, 409, message)
		return
	}

	switch err {
	case nil:
		message, _ := swag.WriteJSON(user)
		sendResponse(w, 201, message)
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}

// /user/{nickname}/profile POST
func UpdateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}
	user := &models.User{}
	err = json.Unmarshal(body, user)
	user.Nickname = nickname

	if err != nil {
		sendResponse(w, 500, []byte(err.Error()))
		return
	}
	err = DB.pool.QueryRow(
		`UPDATE users
		SET fullname = coalesce(nullif($2, ''), fullname),
			email = coalesce(nullif($3, ''), email),
			about = coalesce(nullif($4, ''), about)
		WHERE "nickname" = $1
		RETURNING nickname, fullname, email, about`,
		&user.Nickname,
		&user.Fullname,
		&user.Email,
		&user.About,
	).Scan(
		&user.Nickname,
		&user.Fullname,
		&user.Email,
		&user.About,
	)

	if err != nil {
		if ErrorCode(err) != "" {
			sendResponse(w, 409, []byte(fmt.Sprintf(`{"message": "This email is already registered by user: %s"}`, nickname)))
			return
		}
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find user by nickname: %s"}`, nickname)))
		return

	}

	switch err {
	case nil:
		message, _ := json.Marshal(user)
		sendResponse(w, 200, message)
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}

// /user/{nickname}/profile GET
func GetUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]
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
		sendResponse(w, 404, []byte(fmt.Sprintf(`{"message": "Can't find user by nickname: %s"}`, nickname)))
		return
	}

	switch err {
	case nil:
		message, _ := json.Marshal(user)
		sendResponse(w, 200, message)
	default:
		sendResponse(w, 500, []byte(err.Error()))
	}
}
