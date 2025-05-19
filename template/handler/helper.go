package handler

import (
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"{{projectName}}/web"
)

var (
	PublicFS, _  = fs.Sub(web.WebFs, "template/public")
	PrivateFs, _ = fs.Sub(web.WebFs, "template/private")
)

func renderPublic(w http.ResponseWriter, data any, files ...string) {
	t := template.Must(template.ParseFS(PublicFS, files...))
	if err := t.Execute(w, data); err != nil {
		internal(w)
		return
	}
}

func renderPrivate(w http.ResponseWriter, data any, files ...string) {
	t := template.Must(template.ParseFS(PrivateFs, files...))
	if err := t.Execute(w, data); err != nil {
		internal(w)
		return
	}
}

func unprocessable(w http.ResponseWriter) { w.WriteHeader(http.StatusUnprocessableEntity) }
func unauthorized(w http.ResponseWriter)  { w.WriteHeader(http.StatusUnauthorized) }
func badRequest(w http.ResponseWriter)    { w.WriteHeader(http.StatusBadRequest) }
func internal(w http.ResponseWriter)      { w.WriteHeader(http.StatusInternalServerError) }
func conflict(w http.ResponseWriter)      { w.WriteHeader(http.StatusConflict) }

func hasEmptyString(w http.ResponseWriter, s ...string) bool {
	for _, v := range s {
		if v == "" {
			unprocessable(w)
			return true
		}
	}
	return false
}

func tolowerall(s ...*string) {
	for _, str := range s {
		if *str != "" {
			*str = strings.ToLower(*str)
		}
	}
}
