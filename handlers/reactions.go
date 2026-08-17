package handlers

import (
	"database/sql"
	"net/http"
	"net/url"
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

		action, err := database.SetPostReaction(db, postID, userID, reactionType)
		if err != nil {
			http.Error(w, "could not save reaction", http.StatusInternalServerError)
			return
		}

		msg := reactionMessage(action, reactionType)

		// Check the Referer header to redirect the user back to the page they were on
		redirectURL := r.Header.Get("Referer")
		if redirectURL == "" {
			redirectURL = "/posts/" + idStr
		}
		redirectURL = withFlash(redirectURL, "success", msg)

		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

// reactionMessage builds the toast text based on what actually happened to
// the reaction (added / removed / changed), not just which button the user
// clicked — clicking "like" on a post you already liked removes it, and the
// message needs to reflect that instead of always saying "Post liked".
func reactionMessage(action database.ReactionAction, reactionType models.ReactionType) string {
	verb := "Like"
	if reactionType == models.Dislike {
		verb = "Dislike"
	}

	switch action {
	case database.ReactionRemoved:
		return verb + " Removed"
	case database.ReactionAdded, database.ReactionChanged:
		if reactionType == models.Like {
			return "Post Liked"
		}
		return "Post Disliked"
	default:
		return "Reaction updated"
	}
}

// withFlash sets a flash query param on a URL, replacing any existing
// success/error params rather than appending to them.
func withFlash(rawURL, key, value string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		return rawURL + sep + key + "=" + url.QueryEscape(value)
	}

	q := u.Query()
	q.Del("success")
	q.Del("error")
	q.Set(key, value)
	u.RawQuery = q.Encode()

	return u.String()
}