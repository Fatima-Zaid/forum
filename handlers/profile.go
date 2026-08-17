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
	Filter     string
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

		filter := r.URL.Query().Get("filter")
		if filter != "liked" {
			filter = "created" // default AND normalizes any garbage query value
		}

		var posts []models.Post
		if filter == "liked" {
			posts, err = database.GetPostsLikedByUser(db, currentUser.ID)
		} else {
			posts, err = database.GetPostsByUser(db, currentUser.ID)
		}
		if err != nil {
			log.Println("profile: get posts error:", err)
			RenderError(w, r, http.StatusInternalServerError, "Could not load your posts")
			return
		}

		if err := database.ApplyUserReactions(db, currentUser.ID, posts); err != nil {
			log.Println("profile: apply user reactions error:", err)
			RenderError(w, r, http.StatusInternalServerError, "Could not load your posts")
			return
		}

		// PostCount is meant as "how many posts has this user created" for
		// the header stat — it shouldn't shrink/change when the user is on
		// the "Liked Posts" tab, so don't just reuse len(posts) there.
		postCount := len(posts)
		if filter == "liked" {
			created, err := database.GetPostsByUser(db, currentUser.ID)
			if err != nil {
				log.Println("profile: get post count error:", err)
				RenderError(w, r, http.StatusInternalServerError, "Internal Server Error")
				return
			}
			postCount = len(created)
		}

		renderTemplate(w, r, "profile.html", ProfilePageData{
			Title:      "Profile",
			User:       currentUser,
			IsLoggedIn: true,
			Posts:      posts,
			PostCount:  postCount,
			Filter:     filter,
		})
	}
}