package handlers

import (
	"net/http"

	"forum/utils"
)

// HomeHandler renders the index page, showing logged-in state if applicable.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	user, err := utils.GetUserFromSession(r)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	authRenderTemplate(w, "index.html", pageData{Title: "Home", User: user})
}