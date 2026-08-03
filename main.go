package main

import (
	"log"
	"net/http"

	"forum/database"
)

func main() {
	db, err := database.InitDB("./forum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	_ = db 

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}