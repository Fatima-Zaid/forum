package handlers

import (
	"html/template"
	"log"
	"net/http"
)

func renderTemplate(w http.ResponseWriter, page string, data any) {
	tmpl, err := template.ParseFiles(
		templatesDir+"layout.html",
		templatesDir+page,
		templatesDir+"partials/nav.html",
		templatesDir+"partials/comment.html",
	)
	if err != nil {
		log.Println("template parse error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Println("template exec error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}