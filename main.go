package main

import (
	"log"
	"net/http"

	"forum/database"
	"forum/routes"
)

func main() {
	_, err := database.InitDB("./forum.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	mux := http.NewServeMux()
	routes.RegisterRoutes(mux)

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}