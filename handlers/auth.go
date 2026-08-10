package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"forum/database"
	"forum/models"
	"forum/utils"
)

const templatesDir = "templates/"

// pageData is the data passed into every page template.
type pageData struct {
	Title string
	Error string
	User  *models.User
}

// ---------- Register ----------

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	if user, _ := utils.GetUserFromSession(r); user != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	renderTemplate(w, r, "register.html", pageData{
		Title: "Register",
	})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		renderTemplate(w, r, "register.html", pageData{
			Title: "Register",
			Error: "Invalid form submission",
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if email == "" || username == "" || password == "" {
		renderTemplate(w, r, "register.html", pageData{
			Title: "Register",
			Error: "All fields are required",
		})
		return
	}

	if _, err := mail.ParseAddress(email); err != nil {
		renderTemplate(w, r, "register.html", pageData{
			Title: "Register",
			Error: "Invalid email address",
		})
		return
	}

	if len(password) < 8 {
		renderTemplate(w, r, "register.html", pageData{
			Title: "Register",
			Error: "Password must be at least 8 characters",
		})
		return
	}

	// Check whether the email is already taken.
	var exists int

	if err := database.DB.QueryRow(
		`SELECT COUNT(*) FROM users WHERE email = ?`,
		email,
	).Scan(&exists); err != nil {

		log.Println("db error checking email:", err)

		RenderError(
			w,
			r,
			http.StatusInternalServerError,
			"Internal Server Error",
		)
		return
	}

	if exists > 0 {
		renderTemplate(w, r, "register.html", pageData{
			Title: "Register",
			Error: "Email is already registered",
		})
		return
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		log.Println("hash error:", err)

		RenderError(
			w,
			r,
			http.StatusInternalServerError,
			"Internal Server Error",
		)
		return
	}

	result, err := database.DB.Exec(
		`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
		username,
		email,
		hashedPassword,
	)

	if err != nil {
		log.Println("insert user error:", err)

		RenderError(
			w,
			r,
			http.StatusInternalServerError,
			"Internal Server Error",
		)
		return
	}

	userID, err := result.LastInsertId()
	if err != nil {
		log.Println("lastInsertId error:", err)

		RenderError(
			w,
			r,
			http.StatusInternalServerError,
			"Internal Server Error",
		)
		return
	}

	session, err := utils.CreateSession(int(userID))
	if err != nil {
		log.Println("create session error:", err)

		RenderError(
			w,
			r,
			http.StatusInternalServerError,
			"Internal Server Error",
		)
		return
	}

	utils.SetSessionCookie(w, r, session)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------- Login ----------

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	if user, _ := utils.GetUserFromSession(r); user != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	renderTemplate(w, r, "login.html", pageData{
		Title: "Login",
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	if err := r.ParseForm(); err != nil {
		renderTemplate(w, r, "login.html", pageData{
			Title: "Login",
			Error: "Invalid form submission",
		})
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	if email == "" || password == "" {
		renderTemplate(w, r, "login.html", pageData{
			Title: "Login",
			Error: "Email and password are required",
		})
		return
	}

	var user models.User

	row := database.DB.QueryRow(
		`SELECT id, username, email, password_hash, created_at
		 FROM users
		 WHERE email = ?`,
		email,
	)

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
	)

	if err != nil {
		if err != sql.ErrNoRows {
			log.Println("db error during login:", err)
		}

		// Don't reveal whether the email exists.
		renderTemplate(w, r, "login.html", pageData{
			Title: "Login",
			Error: "Incorrect email or password",
		})
		return
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		renderTemplate(w, r, "login.html", pageData{
			Title: "Login",
			Error: "Incorrect email or password",
		})
		return
	}

	session, err := utils.CreateSession(user.ID)
	if err != nil {
		log.Println("create session error:", err)

		RenderError(
			w,
			r,
			http.StatusInternalServerError,
			"Internal Server Error",
		)
		return
	}

	utils.SetSessionCookie(w, r, session)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ---------- Logout ----------

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RenderError(w, r, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	utils.DestroySession(w, r)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
