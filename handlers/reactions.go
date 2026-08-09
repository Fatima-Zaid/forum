package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"forum/database"
	"forum/models"
)


func ReactToPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := currentUserID(db, r)
		if !ok {
			http.Error(w, "you must be logged in to react", http.StatusUnauthorized)
			return
		}

		// expects path like /posts/12/react
		idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/posts/"), "/react")
		postID, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid post id", http.StatusBadRequest)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}
		reactionType := models.ReactionType(r.FormValue("type"))
		if reactionType != models.Like && reactionType != models.Dislike {
			http.Error(w, "type must be like or dislike", http.StatusBadRequest)
			return
		}

		if err := database.SetPostReaction(db, postID, userID, reactionType); err != nil {
			http.Error(w, "could not save reaction", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/posts/"+idStr, http.StatusSeeOther)
	}
}
