package models

import "net/http"

func GetUserData(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("User data placeholder"))
}
