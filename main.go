package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"forum/database"
	"forum/handlers"
)

func main() {
	db, err := database.InitDB("forum.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer database.Close()

	mux := http.NewServeMux()

	// Static assets (css, uploaded images).
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Posts (this part).
	mux.HandleFunc("/", handlers.ListPostsHandler(db))       // GET  /
	mux.HandleFunc("/posts", handlers.CreatePostHandler(db)) // POST /posts
	mux.HandleFunc("/posts/", postsSubrouter(db))            // GET /posts/{id}, POST /posts/{id}/react

	addr := ":8080"
	log.Println("forum running at http://localhost" + addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func postsSubrouter(db *sql.DB) http.HandlerFunc {
	viewPost := handlers.GetPostHandler(db)
	react := handlers.ReactToPostHandler(db)

	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/react"):
			react(w, r)
		case strings.HasSuffix(r.URL.Path, "/comments"):
			// handlers.CreateCommentHandler(db)(w, r) — wire up once it exists
			http.Error(w, "comments not implemented yet", http.StatusNotImplemented)
		default:
			viewPost(w, r)
		}
	}
}
