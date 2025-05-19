package handler

import "net/http"

func Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("dashboard"))
}
