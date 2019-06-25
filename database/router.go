package database

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func CreateRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/forum/create", CreateForum).Methods("POST")
	r.HandleFunc("/api/forum/{slug}/create", CreateForumThread).Methods("POST")
	r.HandleFunc("/api/forum/{slug}/details", GetForum).Methods("GET")
	r.HandleFunc("/api/forum/{slug}/threads", GetForumThreads).Methods("GET")
	r.HandleFunc("/api/forum/{slug}/users", Getforum_users).Methods("GET")
	r.HandleFunc("/api/user/{nickname}/create", CreateUser).Methods("POST")
	r.HandleFunc("/api/user/{nickname}/profile", GetUser).Methods("GET")
	r.HandleFunc("/api/user/{nickname}/profile", UpdateUser).Methods("POST")
	r.HandleFunc("/api/post/{id}/details", GetPost).Methods("GET")
	r.HandleFunc("/api/post/{id}/details", UpdatePost).Methods("POST")
	r.HandleFunc("/api/thread/{slug_or_id}/details", GetThread).Methods("GET")
	r.HandleFunc("/api/thread/{slug_or_id}/details", UpdateThread).Methods("POST")
	r.HandleFunc("/api/thread/{slug_or_id}/create", CreatePost).Methods("POST")
	r.HandleFunc("/api/service/status", GetStatusDatabase).Methods("GET")
	r.HandleFunc("/api/service/clear", ClearAllDatabase).Methods("POST")
	r.HandleFunc("/api/thread/{slug_or_id}/posts", GetThreadPosts).Methods("GET")
	r.HandleFunc("/api/thread/{slug_or_id}/vote", MakeThreadVote).Methods("POST")

	return r
}

func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now().Second()
		next.ServeHTTP(w, r)
		t2 := time.Now().Second()
		dt := t2 - t1
		// выводим медленные запросы
		if dt >= 1 {
			log.Println(
				dt,
				r.Method,
				r.URL.Path,
				"limit", r.URL.Query().Get("limit"),
				"since", r.URL.Query().Get("since"),
				"desc", r.URL.Query().Get("desc"),
				"sort", r.URL.Query().Get("sort"),
			)
		}
	})
}
