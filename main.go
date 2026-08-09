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
	mux.HandleFunc("/", handlers.ListPostsHandler(db))                // GET  /
	mux.HandleFunc("/posts", handlers.CreatePostHandler(db))          // POST /posts
	mux.HandleFunc("/posts/", postsSubrouter(db))                     // GET /posts/{id}, POST /posts/{id}/react, GET+POST /posts/{id}/edit



	addr := ":8080"
	log.Println("forum running at http://localhost" + addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// postsSubrouter dispatches everything under /posts/{id} since the stdlib
// mux here doesn't do path-parameter matching. Routes on suffix:
//
//	GET  /posts/{id}          -> view post
//	GET+POST /posts/{id}/edit -> edit post (form + save)
//	POST /posts/{id}/react    -> like/dislike
//	POST /posts/{id}/delete   -> delete post
//	POST /posts/{id}/comments -> add comment (your partner's handler, not wired yet)
func postsSubrouter(db *sql.DB) http.HandlerFunc {
	viewPost := handlers.GetPostHandler(db)
	editPost := handlers.EditPostHandler(db)
	react := handlers.ReactToPostHandler(db)
	deletePost := handlers.DeletePostHandler(db)

	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/edit"):
			editPost(w, r)
		case strings.HasSuffix(r.URL.Path, "/react"):
			react(w, r)
		case strings.HasSuffix(r.URL.Path, "/delete"):
			deletePost(w, r)
		case strings.HasSuffix(r.URL.Path, "/comments"):
			// handlers.CreateCommentHandler(db)(w, r) — wire up once it exists
			http.Error(w, "comments not implemented yet", http.StatusNotImplemented)
		default:
			viewPost(w, r)
		}
	}
}