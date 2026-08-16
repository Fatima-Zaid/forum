package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"forum/utils"
)

func renderTemplate(w http.ResponseWriter, r *http.Request, page string, data any) {
	funcMap := template.FuncMap{
		"flashSuccess": func() string { return r.URL.Query().Get("success") },
		"flashError":   func() string { return r.URL.Query().Get("error") },
	}

	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFiles(
		templatesDir+"layout.html",
		templatesDir+page,
		templatesDir+"partials/nav.html",
		templatesDir+"partials/comment.html",
	)
	if err != nil {
		log.Println("template parse error:", err)
		RenderError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Println("template exec error:", err)
		RenderError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}
}

// RenderError renders the custom error page.
func RenderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	currentUser, _ := utils.GetUserFromSession(r)

	data := struct {
		Title   string
		Code    int
		Message string
		User    interface{}
	}{
		Title:   "Error",
		Code:    status,
		Message: message,
		User:    currentUser,
	}

	funcMap := template.FuncMap{
		"flashSuccess": func() string { return r.URL.Query().Get("success") },
		"flashError":   func() string { return r.URL.Query().Get("error") },
	}

	tmpl, err := template.New("layout.html").Funcs(funcMap).ParseFiles(
		templatesDir+"layout.html",
		templatesDir+"error.html",
	)
	if err != nil {
		log.Println("error template parse error:", err)
		http.Error(w, message, status)
		return
	}

	// Render the template into memory first.
	// This prevents us from sending the HTTP status before
	// we know that the template actually works.
	var buf bytes.Buffer

	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Println("error template exec error:", err)
		http.Error(w, message, status)
		return
	}

	// Only send the status after the template rendered successfully.
	w.WriteHeader(status)

	_, err = w.Write(buf.Bytes())
	if err != nil {
		log.Println("error response write error:", err)
	}
}