package models

import "net/http"

func GetdbData(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("User data placeholder"))
}
