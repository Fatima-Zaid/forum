package handlers

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"forum/database"
	"forum/models"
)


type IndexPageData struct {
	Posts          []models.Post
	Categories     []models.Category
	IsLoggedIn     bool
	ActiveFilter   string // "", "created", "liked"
	ActiveCategory int    // 0 means none selected
}

// PostPageData is what post.html renders: one post plus its comments.
type PostPageData struct {
	Post       *models.Post
	Comments   []models.Comment
	IsLoggedIn bool
}


func currentUserID(db *sql.DB, r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return 0, false
	}
	session, err := database.GetValidSession(db, cookie.Value)
	if err != nil {
		return 0, false
	}
	return session.UserID, true
}

// CreatePostHandler handles POST /posts. Registered users only.
func CreatePostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := currentUserID(db, r)
		if !ok {
			http.Error(w, "you must be logged in to post", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form data", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		gameTitle := strings.TrimSpace(r.FormValue("game_title"))
		content := strings.TrimSpace(r.FormValue("content"))

		if title == "" || content == "" {
			http.Error(w, "title and content are required", http.StatusBadRequest)
			return
		}

		// Existing categories are submitted as checked IDs: <input name="category_ids" value="3">
		var categoryIDs []int
		for _, raw := range r.Form["category_ids"] {
			id, err := strconv.Atoi(raw)
			if err != nil {
				http.Error(w, "invalid category id", http.StatusBadRequest)
				return
			}
			categoryIDs = append(categoryIDs, id)
		}

	
		if newCats := strings.TrimSpace(r.FormValue("new_categories")); newCats != "" {
			for _, name := range strings.Split(newCats, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				id, err := database.GetOrCreateCategory(db, name)
				if err != nil {
					http.Error(w, "could not create category", http.StatusInternalServerError)
					return
				}
				categoryIDs = append(categoryIDs, id)
			}
		}

		if len(categoryIDs) == 0 {
			http.Error(w, "select at least one category", http.StatusBadRequest)
			return
		}

		postID, err := database.CreatePost(db, userID, title, gameTitle, content, categoryIDs)
		if err != nil {
			http.Error(w, "could not create post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/posts/"+strconv.Itoa(postID), http.StatusSeeOther)
	}
}

func ListPostsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		categories, err := database.GetAllCategories(db)
		if err != nil {
			http.Error(w, "could not load categories", http.StatusInternalServerError)
			return
		}

		userID, isLoggedIn := currentUserID(db, r)
		filter := r.URL.Query().Get("filter") // "created" or "liked"
		categoryParam := r.URL.Query().Get("category")

		data := IndexPageData{
			Categories: categories,
			IsLoggedIn: isLoggedIn,
		}

		switch {
		case filter == "created" || filter == "liked":
			if !isLoggedIn {
				http.Error(w, "login required for this filter", http.StatusUnauthorized)
				return
			}
			data.ActiveFilter = filter
			var posts []models.Post
			var err error
			if filter == "created" {
				posts, err = database.GetPostsByUser(db, userID)
			} else {
				posts, err = database.GetPostsLikedByUser(db, userID)
			}
			if err != nil {
				http.Error(w, "could not load posts", http.StatusInternalServerError)
				return
			}
			data.Posts = posts

		case categoryParam != "":
			catID, convErr := strconv.Atoi(categoryParam)
			if convErr != nil {
				http.Error(w, "invalid category", http.StatusBadRequest)
				return
			}
			posts, err := database.GetPostsByCategory(db, catID)
			if err != nil {
				http.Error(w, "could not load posts", http.StatusInternalServerError)
				return
			}
			data.ActiveCategory = catID
			data.Posts = posts

		default:
			posts, err := database.GetAllPosts(db)
			if err != nil {
				http.Error(w, "could not load posts", http.StatusInternalServerError)
				return
			}
			data.Posts = posts
		}

		renderTemplate(w, "index.html", data)
	}
}

// GetPostHandler handles GET /posts/{id}. Visible to everyone, logged in or not.
func GetPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		idStr := strings.TrimPrefix(r.URL.Path, "/posts/")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid post id", http.StatusBadRequest)
			return
		}

		post, err := database.GetPostByID(db, id)
		if err == database.ErrNotFound {
			http.Error(w, "post not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "could not load post", http.StatusInternalServerError)
			return
		}

		comments, err := database.GetCommentsByPost(db, id)
		if err != nil {
			http.Error(w, "could not load comments", http.StatusInternalServerError)
			return
		}

		_, isLoggedIn := currentUserID(db, r)

		renderTemplate(w, "post.html", PostPageData{
			Post:       post,
			Comments:   comments,
			IsLoggedIn: isLoggedIn,
		})
	}
}


func renderTemplate(w http.ResponseWriter, name string, data any) {
	tmpl, err := template.ParseFiles(
		"templates/layout.html",
		"templates/"+name,
		"templates/partials/nav.html",
		"templates/partials/comment.html",
	)
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}