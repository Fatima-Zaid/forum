package routes

import (
	"database/sql"
	"net/http"
	"strings"

	"forum/handlers"
)

// RegisterRoutes wires up every route in the app onto mux. db is needed
// here (not just in main) because the posts/reactions handlers are
// constructed as closures over it (e.g. handlers.ListPostsHandler(db)).
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {
	// static files (css, js, uploaded images) served from ./static
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// posts own "/" now: it lists posts and handles the category/created/liked
	// filters required by the spec. The old plain-welcome homeHandler is gone —
	// heads up to your friend since this route used to be hers.
	mux.HandleFunc("/", handlers.ListPostsHandler(db))
	mux.HandleFunc("/posts", handlers.CreatePostHandler(db))
	mux.HandleFunc("/posts/", postsSubrouter(db))

	// auth routes
	mux.HandleFunc("/register", registerRouter)
	mux.HandleFunc("/login", loginRouter)
	mux.HandleFunc("/logout", handlers.LogoutHandler)
}

// postsSubrouter dispatches everything under /posts/{id} since the stdlib
// mux here doesn't do path-parameter matching. Routes on suffix:
//
//	GET  /posts/{id}          -> view post
//	GET+POST /posts/{id}/edit -> edit post (form + save)
//	POST /posts/{id}/react    -> like/dislike
//	POST /posts/{id}/delete   -> delete post
//	POST /posts/{id}/comments -> add comment (wire up once your comment
//	                             handler exists — see TODO below)
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
			// TODO: swap in your comment handler once it exists, e.g.
			// handlers.CreateCommentHandler(db)(w, r)
			http.Error(w, "comments not implemented yet", http.StatusNotImplemented)
		default:
			viewPost(w, r)
		}
	}
}

func registerRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		handlers.RegisterPageHandler(w, r)
		return
	}
	handlers.RegisterHandler(w, r)
}

func loginRouter(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		handlers.LoginPageHandler(w, r)
		return
	}
	handlers.LoginHandler(w, r)
}