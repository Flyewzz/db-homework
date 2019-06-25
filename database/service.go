package database

import (
	"encoding/json"
	"net/http"

	"github.com/Flyewzz/db-homework/models"
)

// /service/clear POST
func ClearAllDatabase(w http.ResponseWriter, r *http.Request) {
	DB.pool.Exec(`
		TRUNCATE threads, users, votes, forumusers, forums, posts;
	`)
	SendResponce(w, 200, []byte("Database cleared successfully!"))
}

// /service/status GET
func GetStatusDatabase(w http.ResponseWriter, r *http.Request) {
	status := &models.Status{}
	DB.pool.QueryRow(
		`SELECT
		(SELECT COUNT(*) FROM forums) AS forums,
		(SELECT COUNT(*) FROM posts) AS posts,
		(SELECT COALESCE(SUM(threads), 0) FROM forums WHERE threads > 0) AS threads,
		(SELECT COUNT(*) FROM users) AS users;`,
	).Scan(
		&status.Forum,
		&status.Post,
		&status.Thread,
		&status.User,
	)

	message, err := json.Marshal(status)
	if err == nil {
		makeResponse(w, 200, message)
		return
	}
	makeResponse(w, 500, []byte(err.Error()))
}
