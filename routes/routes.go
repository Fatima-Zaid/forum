package routes

import (
	"net/http"

	"forum/handlers"
)


func RegisterRoutes(mux *http.ServeMux) {
	// static files (css, js, images) served from ./static
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// home page
	mux.HandleFunc("/", homeHandler)

	// auth routes
	mux.HandleFunc("/register", registerRouter)
	mux.HandleFunc("/login", loginRouter)
	mux.HandleFunc("/logout", handlers.LogoutHandler)
}


func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	handlers.HomeHandler(w, r)
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