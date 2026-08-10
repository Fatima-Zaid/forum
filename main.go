package main

import (
	"log"
	"net/http"

	"forum/database"
	"forum/routes"
)

func main() {

	db, err := database.InitDB("forum.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	defer database.Close()

	mux := http.NewServeMux()

	routes.RegisterRoutes(mux, db)

	addr := ":8080"

	log.Println("forum running at http://localhost" + addr)

	log.Fatal(http.ListenAndServe(addr, mux))
}
