package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"forum/database"
	"forum/models"
	"forum/utils"
)

type IndexPageData struct {
	Title          string
	Posts          []models.Post
	Categories     []models.Category
	IsLoggedIn     bool
	ActiveFilter   string
	ActiveCategory int
	User           *models.User
}

type NewPostPageData struct {
	Title      string
	Categories []models.Category
	IsLoggedIn bool
	User       *models.User
	Error      string
}

type PostPageData struct {
	Title         string
	Post          *models.Post
	Comments      []models.Comment
	IsLoggedIn    bool
	CurrentUserID int
	User          *models.User
}

type EditPostPageData struct {
	Title      string
	Post       *models.Post
	Categories []models.Category
	IsLoggedIn bool
	User       *models.User
	Error      string
}

func NewPostPageHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		currentUser, err := utils.GetUserFromSession(r)
		if err != nil {
			RenderError(w, r, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if currentUser == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		categories, err := database.GetAllCategories(db)
		if err != nil {
			log.Println("new post: get categories error:", err)
			RenderError(w, r, http.StatusInternalServerError, "Could not load categories")
			return
		}

		renderTemplate(w, r, "new.html", NewPostPageData{
			Title:      "New Post - ",
			Categories: categories,
			IsLoggedIn: true,
			User:       currentUser,
		})
	}
}

func EditPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := currentUserID(db, r)
		if !ok {
			RenderError(w, r, http.StatusUnauthorized, "You must be logged in")
			return
		}

		idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/posts/"), "/edit")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			RenderError(w, r, http.StatusBadRequest, "Invalid Post ID")
			return
		}

		post, err := database.GetPostByID(db, id)
		if err == database.ErrNotFound {
			RenderError(w, r, http.StatusNotFound, "Post Not Found")
			return
		}
		if err != nil {
			log.Println("edit: get post error:", err)
			RenderError(w, r, http.StatusInternalServerError, "Could not load post")
			return
		}
		if post.UserID != userID {
			RenderError(w, r, http.StatusForbidden, "You do not own this post")
			return
		}

		switch r.Method {
		case http.MethodGet:
			categories, err := database.GetAllCategories(db)
			if err != nil {
				log.Println("edit: get categories error:", err)
				RenderError(w, r, http.StatusInternalServerError, "Could not load categories")
				return
			}
			currentUser, _ := utils.GetUserFromSession(r)
			renderTemplate(w, r, "edit.html", EditPostPageData{
				Post:       post,
				Categories: categories,
				IsLoggedIn: true,
				User:       currentUser,
			})

		case http.MethodPost:
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				redirectWithError(w, r, "/posts/"+idStr+"/edit", "Invalid form data (image too large?)")
				return
			}

			title := strings.TrimSpace(r.FormValue("title"))
			gameTitle := strings.TrimSpace(r.FormValue("game_title"))
			content := strings.TrimSpace(r.FormValue("content"))
			if title == "" || content == "" {
				redirectWithError(w, r, "/posts/"+idStr+"/edit", "Title and content are required")
				return
			}
			imageURL, err := saveUploadedImage(r, "image", "posts")
			if err != nil {
				redirectWithError(w, r, "/posts/"+idStr+"/edit", err.Error())
				return
			}

			// If no file was uploaded, use the URL instead.
			if imageURL == "" {
				imageURL = strings.TrimSpace(r.FormValue("image_url"))
			}

			var categoryIDs []int
			for _, raw := range r.Form["category_ids"] {
				cid, err := strconv.Atoi(raw)
				if err != nil {
					redirectWithError(w, r, "/posts/"+idStr+"/edit", "Invalid category selected")
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
						RenderError(w, r, http.StatusInternalServerError, "Could not create category")
						return
					}
					categoryIDs = append(categoryIDs, cid)
				}
			}
			if len(categoryIDs) == 0 {
				redirectWithError(w, r, "/posts/"+idStr+"/edit", "Select at least one category")
				return
			}

			if err := database.UpdatePost(db, id, userID, title, gameTitle, content, imageURL, categoryIDs); err != nil {
				log.Println("edit: update post error:", err)
				RenderError(w, r, http.StatusInternalServerError, "Could not update post")
				return
			}

			http.Redirect(w, r, "/posts/"+idStr+"?success="+url.QueryEscape("Post updated successfully"), http.StatusSeeOther)

		default:
			RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
		}
	}
}

func currentUserID(db *sql.DB, r *http.Request) (int, bool) {
	user, err := utils.GetUserFromSession(r)
	if err != nil || user == nil {
		return 0, false
	}
	return user.ID, true
}

const maxUploadSize = 10 << 20 // 10 MB

func redirectWithError(w http.ResponseWriter, r *http.Request, path, msg string) {
	http.Redirect(w, r, path+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
}

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

func CreatePostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		userID, ok := currentUserID(db, r)
		if !ok {
			RenderError(w, r, http.StatusUnauthorized, "You must be logged in to post")
			return
		}

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			redirectWithError(w, r, "/posts/new", "Invalid form data (image too large?)")
			return
		}

		title := strings.TrimSpace(r.FormValue("title"))
		gameTitle := strings.TrimSpace(r.FormValue("game_title"))
		content := strings.TrimSpace(r.FormValue("content"))

		if title == "" || content == "" {
			redirectWithError(w, r, "/posts/new", "Title and content are required")
			return
		}

		imageURL, err := saveUploadedImage(r, "image", "posts")
		if err != nil {
			redirectWithError(w, r, "/posts/new", err.Error())
			return
		}

		// If no file was uploaded, use the URL instead.
		if imageURL == "" {
			imageURL = strings.TrimSpace(r.FormValue("image_url"))
		}

		var categoryIDs []int
		for _, raw := range r.Form["category_ids"] {
			id, err := strconv.Atoi(raw)
			if err != nil {
				redirectWithError(w, r, "/posts/new", "Invalid category selected")
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
					log.Println("create post: category error:", err)
					RenderError(w, r, http.StatusInternalServerError, "Could not create category")
					return
				}
				categoryIDs = append(categoryIDs, id)
			}
		}

		if len(categoryIDs) == 0 {
			redirectWithError(w, r, "/posts/new", "Select at least one category")
			return
		}

		postID, err := database.CreatePost(db, userID, title, gameTitle, content, imageURL, categoryIDs)
		if err != nil {
			log.Println("create post: db error:", err)
			redirectWithError(w, r, "/posts/new", "Something went wrong, post was not created")
			return
		}

		http.Redirect(w, r, "/posts/"+strconv.Itoa(postID)+"?success="+url.QueryEscape("Post created successfully"), http.StatusSeeOther)
	}
}

func DeletePostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		userID, ok := currentUserID(db, r)
		if !ok {
			RenderError(w, r, http.StatusUnauthorized, "You must be logged in")
			return
		}

		idStr := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/posts/"), "/delete")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			RenderError(w, r, http.StatusBadRequest, "Invalid Post ID")
			return
		}

		if err := database.DeletePost(db, id, userID); err == database.ErrNotFound {
			RenderError(w, r, http.StatusForbidden, "Post not found or you don't own it")
			return
		} else if err != nil {
			RenderError(w, r, http.StatusInternalServerError, "Could not delete post")
			return
		}

		http.Redirect(w, r, "/?success="+url.QueryEscape("Post deleted"), http.StatusSeeOther)
	}
}

func ListPostsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strict check for non-existent paths hitting "/"
		if r.URL.Path != "/" {
			RenderError(w, r, http.StatusNotFound, "Page Not Found")
			return
		}

		if r.Method != http.MethodGet {
			RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		categories, err := database.GetAllCategories(db)
		if err != nil {
			log.Println("index: get categories error:", err)
			RenderError(w, r, http.StatusInternalServerError, "Could not load categories")
			return
		}

		userID, isLoggedIn := currentUserID(db, r)
		currentUser, _ := utils.GetUserFromSession(r)
		filter := r.URL.Query().Get("filter")
		categoryParam := r.URL.Query().Get("category")

		data := IndexPageData{
			Categories: categories,
			IsLoggedIn: isLoggedIn,
			User:       currentUser,
		}

		switch {
		case filter == "created" || filter == "liked":
			if !isLoggedIn {
				RenderError(w, r, http.StatusUnauthorized, "Login required for this filter")
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
				RenderError(w, r, http.StatusInternalServerError, "Could not load posts")
				return
			}
			data.Posts = posts

		case categoryParam != "":
			catID, convErr := strconv.Atoi(categoryParam)
			if convErr != nil {
				RenderError(w, r, http.StatusBadRequest, "Invalid Category Selection")
				return
			}
			posts, err := database.GetPostsByCategory(db, catID)
			if err != nil {
				log.Println("index: get posts by category error:", err)
				RenderError(w, r, http.StatusInternalServerError, "Could not load posts")
				return
			}
			data.ActiveCategory = catID
			data.Posts = posts

		default:
			posts, err := database.GetAllPosts(db)
			if err != nil {
				log.Println("index: get all posts error:", err)
				RenderError(w, r, http.StatusInternalServerError, "Could not load posts")
				return
			}
			data.Posts = posts
		}

		renderTemplate(w, r, "index.html", data)
	}
}

func GetPostHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodGet {
			RenderError(
				w,
				r,
				http.StatusMethodNotAllowed,
				"Method Not Allowed",
			)
			return
		}

		// Remove /posts/ from the beginning.
		idStr := strings.TrimPrefix(r.URL.Path, "/posts/")

		// Convert the remaining part to an integer.
		id, err := strconv.Atoi(idStr)
		if err != nil {
			RenderError(
				w,
				r,
				http.StatusBadRequest,
				"Invalid Post ID",
			)
			return
		}

		// Get the post from the database.
		post, err := database.GetPostByID(db, id)

		// The database function converts sql.ErrNoRows
		// into database.ErrNotFound.
		if err == database.ErrNotFound {
			RenderError(
				w,
				r,
				http.StatusNotFound,
				"Post Not Found",
			)
			return
		}

		// Any other database error = 500.
		if err != nil {
			log.Println("post: get post error:", err)

			RenderError(
				w,
				r,
				http.StatusInternalServerError,
				"Could not load post",
			)
			return
		}

		comments, err := database.GetCommentsByPost(db, id)
		if err != nil {
			log.Println("post: get comments error:", err)

			RenderError(
				w,
				r,
				http.StatusInternalServerError,
				"Could not load comments",
			)
			return
		}

		currentUID, isLoggedIn := currentUserID(db, r)
		currentUser, _ := utils.GetUserFromSession(r)

		renderTemplate(
			w,
			r,
			"post.html",
			PostPageData{
				Title:         post.Title,
				Post:          post,
				Comments:      comments,
				IsLoggedIn:    isLoggedIn,
				CurrentUserID: currentUID,
				User:          currentUser,
			},
		)
	}
}
