package database

import (
	"net/http"
)

func sendResponse(w http.ResponseWriter, statusCode int, message []byte) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(message)
}
