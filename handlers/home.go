package handlers

import (
	"net/http"

	"forum/utils"
)

// HomeHandler renders the index page, showing logged-in state if applicable.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		RenderError(w, r, http.StatusNotFound, "Page Not Found")
		return
	}

	user, err := utils.GetUserFromSession(r)
	if err != nil {
		RenderError(w, r, http.StatusBadRequest, "Invalid Category Selection")
		return
	}
	renderTemplate(w, r, "index.html", pageData{Title: "Home", User: user})
}
