package handlers

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"forum/database"
	"forum/models"
)

// IndexPageData is what index.html renders: the post list plus enough
// context (categories, login state, which filter is active) to draw the
// filter nav and the "new post" form.
type IndexPageData struct {
	Posts          []models.Post
	Categories     []models.Category
	IsLoggedIn     bool
	ActiveFilter   string // "", "created", "liked"
	ActiveCategory int    // 0 means none selected
}

// PostPageData is what post.html renders: one post plus its comments.
// CurrentUserID lets the template show the delete button only to the
// post's own author (0 when logged out).
type PostPageData struct {
	Post          *models.Post
	Comments      []models.Comment
	IsLoggedIn    bool
	CurrentUserID int
}

type EditPostPageData struct {
	Post       *models.Post
	Categories []models.Category
	IsLoggedIn bool
}

// EditPostHandler handles both GET /posts/{id}/edit (show the form) and
// POST /posts/{id}/edit (save changes). Only the post's author may edit —
// checked here, and again by UpdatePost's WHERE clause.
func EditPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(db, r)
		if !ok {
			http.Error(w, "you must be logged in", http.StatusUnauthorized)
			return
		}

		idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/posts/"), "/edit")
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
			log.Println("edit: get post error:", err)
			http.Error(w, "could not load post", http.StatusInternalServerError)
			return
		}
		if post.UserID != userID {
			http.Error(w, "you don't own this post", http.StatusForbidden)
			return
		}

		switch r.Method {
		case http.MethodGet:
			categories, err := database.GetAllCategories(db)
			if err != nil {
				log.Println("edit: get categories error:", err)
				http.Error(w, "could not load categories", http.StatusInternalServerError)
				return
			}
			renderTemplate(w, "edit.html", EditPostPageData{
				Post:       post,
				Categories: categories,
				IsLoggedIn: true,
			})

		case http.MethodPost:
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				http.Error(w, "invalid form data (image too large?)", http.StatusBadRequest)
				return
			}

			title := strings.TrimSpace(r.FormValue("title"))
			gameTitle := strings.TrimSpace(r.FormValue("game_title"))
			content := strings.TrimSpace(r.FormValue("content"))
			if title == "" || content == "" {
				http.Error(w, "title and content are required", http.StatusBadRequest)
				return
			}

			// "" here means "no new file picked" — UpdatePost keeps the old image.
			imageURL, err := saveUploadedImage(r, "image", "posts")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var categoryIDs []int
			for _, raw := range r.Form["category_ids"] {
				cid, err := strconv.Atoi(raw)
				if err != nil {
					http.Error(w, "invalid category id", http.StatusBadRequest)
					return
				}
				categoryIDs = append(categoryIDs, cid)
			}
			if newCats := strings.TrimSpace(r.FormValue("new_categories")); newCats != "" {
				for _, name := range strings.Split(newCats, ",") {
					name = strings.TrimSpace(name)
					if name == "" {
						continue
					}
					cid, err := database.GetOrCreateCategory(db, name)
					if err != nil {
						log.Println("edit: create category error:", err)
						http.Error(w, "could not create category", http.StatusInternalServerError)
						return
					}
					categoryIDs = append(categoryIDs, cid)
				}
			}
			if len(categoryIDs) == 0 {
				http.Error(w, "select at least one category", http.StatusBadRequest)
				return
			}

			if err := database.UpdatePost(db, id, userID, title, gameTitle, content, imageURL, categoryIDs); err != nil {
				log.Println("edit: update post error:", err)
				http.Error(w, "could not update post", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/posts/"+idStr, http.StatusSeeOther)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// currentUserID reads the session cookie and validates it directly.
// TEMPORARY: once your partner's auth middleware sets the user in request
// context, replace this with reading from context instead, and delete this
// function. Kept local to this file so it's a one-place swap.
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

const maxUploadSize = 10 << 20 // 10 MB

// saveUploadedImage reads an optional file from a multipart form field and
// saves it under static/uploads/{subdir}/. Returns "" (no error) if the
// field was left empty — image is optional. Returns a URL path starting
// with /static/... suitable for storing directly in the DB and using in
// <img src>.
func saveUploadedImage(r *http.Request, field, subdir string) (string, error) {
	file, header, err := r.FormFile(field)
	if err == http.ErrMissingFile {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		// allowed
	default:
		return "", fmt.Errorf("unsupported image type %q", ext)
	}

	dir := filepath.Join("static", "uploads", subdir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dstPath := filepath.Join(dir, filename)

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return "/static/uploads/" + subdir + "/" + filename, nil
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

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, "invalid form data (image too large?)", http.StatusBadRequest)
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		gameTitle := strings.TrimSpace(r.FormValue("game_title"))
		content := strings.TrimSpace(r.FormValue("content"))

		if title == "" || content == "" {
			http.Error(w, "title and content are required", http.StatusBadRequest)
			return
		}

		imageURL, err := saveUploadedImage(r, "image", "posts")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

		// Optional: user typed a category that isn't in the seeded list.
		// Comma-separated, e.g. "Roguelike, Metroidvania".
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

		postID, err := database.CreatePost(db, userID, title, gameTitle, content, imageURL, categoryIDs)
		if err != nil {
			http.Error(w, "could not create post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/posts/"+strconv.Itoa(postID), http.StatusSeeOther)
	}
}

// DeletePostHandler handles POST /posts/{id}/delete. Only the post's author
// may delete it — enforced both here and again at the DB layer.
func DeletePostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, ok := currentUserID(db, r)
		if !ok {
			http.Error(w, "you must be logged in", http.StatusUnauthorized)
			return
		}

		idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/posts/"), "/delete")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid post id", http.StatusBadRequest)
			return
		}

		if err := database.DeletePost(db, id, userID); err == database.ErrNotFound {
			http.Error(w, "post not found or you don't own it", http.StatusForbidden)
			return
		} else if err != nil {
			http.Error(w, "could not delete post", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// ListPostsHandler handles GET /  (and GET /?category=&filter=).
// Supports the three required filters: category, created (mine), liked.
// "created" and "liked" require login and always refer to the logged-in user.
func ListPostsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		categories, err := database.GetAllCategories(db)
		if err != nil {
			log.Println("index: get categories error:", err)
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
				log.Println("index: get filtered posts error:", err)
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
				log.Println("index: get posts by category error:", err)
				http.Error(w, "could not load posts", http.StatusInternalServerError)
				return
			}
			data.ActiveCategory = catID
			data.Posts = posts

		default:
			posts, err := database.GetAllPosts(db)
			if err != nil {
				log.Println("index: get all posts error:", err)
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
			log.Println("post: get post error:", err)
			http.Error(w, "could not load post", http.StatusInternalServerError)
			return
		}

		comments, err := database.GetCommentsByPost(db, id)
		if err != nil {
			log.Println("post: get comments error:", err)
			http.Error(w, "could not load comments", http.StatusInternalServerError)
			return
		}

		currentUID, isLoggedIn := currentUserID(db, r)

		renderTemplate(w, "post.html", PostPageData{
			Post:          post,
			Comments:      comments,
			IsLoggedIn:    isLoggedIn,
			CurrentUserID: currentUID,
		})
	}
}

// --- rendering helpers ---
// Minimal html/template wiring so this file runs standalone. If you already
// have a shared render helper elsewhere (e.g. in main.go), delete these two
// functions and call your existing one instead.

func renderTemplate(w http.ResponseWriter, name string, data any) {
	tmpl, err := template.ParseFiles(
		"templates/layout.html",
		"templates/"+name,
		"templates/partials/nav.html",
		"templates/partials/comment.html",
	)
	if err != nil {
		log.Println("template parse error:", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Println("template execute error:", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}