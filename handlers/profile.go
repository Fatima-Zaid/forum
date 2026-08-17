package handlers

import (
	"database/sql"
	"log"
	"net/http"

	"forum/database"
	"forum/models"
	"forum/utils"
)

type ProfilePageData struct {
	Title      string
	User       *models.User
	IsLoggedIn bool
	Posts      []models.Post
	PostCount  int
}

// ProfilePageHandler renders the logged-in user's own profile page:
// account info plus the posts they've created. Login required.
func ProfilePageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/profile" {
			RenderError(w, r, http.StatusNotFound, "Page Not Found")
			return
		}
		if r.Method != http.MethodGet {
			RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		currentUser, err := utils.GetUserFromSession(r)
		if err != nil {
			log.Println("profile: get session error:", err)
			RenderError(w, r, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if currentUser == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		posts, err := database.GetPostsByUser(db, currentUser.ID)
		if err != nil {
			log.Println("profile: get posts error:", err)
			RenderError(w, r, http.StatusInternalServerError, "Could not load your posts")
			return
		}

		renderTemplate(w, r, "profile.html", ProfilePageData{
			Title:      "Profile",
			User:       currentUser,
			IsLoggedIn: true,
			Posts:      posts,
			PostCount:  len(posts),
		})
	}
}