package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"./database"
)

func main() {

	database.DB.Connect()
	router := database.

	srv := &http.Server{
		Handler:      router,
		Addr:         ":5000",
		WriteTimeout: 360 * time.Second,
		ReadTimeout:  360 * time.Second,
	}
	fmt.Println("Starting server (port=5000)")
	log.Fatal(srv.ListenAndServe())
}
