package routes

import (
	"database/sql"
	"net/http"
	"strings"

	"forum/handlers"
)

// RegisterRoutes wires up every route in the application.
func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {

	// Static files
	mux.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			http.FileServer(http.Dir("static")),
		),
	)

	// IMPORTANT:
	// "/" is the catch-all route in net/http ServeMux.
	//
	// Therefore requests such as:
	//   /does-not-exist
	//   /hello
	//   /abc/xyz
	//
	// will reach ListPostsHandler, which checks the path
	// and renders our custom 404 page.
	mux.HandleFunc("/", handlers.ListPostsHandler(db))

	// Create post
	mux.HandleFunc("/posts", handlers.CreatePostHandler(db))

	// Everything under /posts/
	mux.HandleFunc("/posts/", postsSubrouter(db))

	// Authentication
	mux.HandleFunc("/register", registerRouter)
	mux.HandleFunc("/login", loginRouter)
	mux.HandleFunc("/logout", handlers.LogoutHandler)

	// Profile 
	mux.HandleFunc("/profile", handlers.ProfilePageHandler(db))
}

// postsSubrouter dispatches everything under /posts/.
func postsSubrouter(db *sql.DB) http.HandlerFunc {

	viewPost := handlers.GetPostHandler(db)
	editPost := handlers.EditPostHandler(db)
	react := handlers.ReactToPostHandler(db)
	deletePost := handlers.DeletePostHandler(db)
	deleteConfirm := handlers.DeleteConfirmPageHandler(db)
	newPostPage := handlers.NewPostPageHandler(db)

	return func(w http.ResponseWriter, r *http.Request) {

		switch {

		// GET /posts/new
		case r.URL.Path == "/posts/new":
			newPostPage(w, r)
			return

		// /posts/{id}/edit
		case strings.HasSuffix(r.URL.Path, "/edit"):
			editPost(w, r)
			return

		// /posts/{id}/react
		case strings.HasSuffix(r.URL.Path, "/react"):
			react(w, r)
			return

		// /posts/{id}/delete-confirm  (must be checked BEFORE /delete,
		// since "/delete-confirm" also ends with "-confirm", not "/delete")
		case strings.HasSuffix(r.URL.Path, "/delete-confirm"):
			deleteConfirm(w, r)
			return

		// /posts/{id}/delete
		case strings.HasSuffix(r.URL.Path, "/delete"):
			deletePost(w, r)
			return

		// /posts/{id}/comments
		case strings.HasSuffix(r.URL.Path, "/comments"):
			http.Error(
				w,
				"comments not implemented yet",
				http.StatusNotImplemented,
			)
			return

		// Everything else under /posts/
		default:
			viewPost(w, r)
			return
		}
	}
}

func registerRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.RegisterPageHandler(w, r)

	case http.MethodPost:
		handlers.RegisterHandler(w, r)

	default:
		handlers.RenderError(
			w,
			r,
			http.StatusMethodNotAllowed,
			"Method Not Allowed",
		)
	}
}

func loginRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.LoginPageHandler(w, r)

	case http.MethodPost:
		handlers.LoginHandler(w, r)

	default:
		handlers.RenderError(
			w,
			r,
			http.StatusMethodNotAllowed,
			"Method Not Allowed",
		)
	}
}