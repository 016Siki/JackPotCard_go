package handlers

import (
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// GetPlayersForGameHandler は指定された room_id に所属するプレイヤー一覧を取得して返すハンドラ。
func GetPlayersForGameHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ---- URLパラメータから room_id を取得 ----
		vars := mux.Vars(r)
		roomIDStr := vars["room_id"]

		roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid room_id", http.StatusBadRequest)
			return
		}
		// ---- DBからプレイヤー一覧を取得 ----
		players, err := models.GetPlayersForGame(db, roomID)
		if err != nil {
			log.Println("GetPlayersForGame error:", err)
			http.Error(w, "Failed to get players", http.StatusInternalServerError)
			return
		}
		// ---- JSONレスポンス返却 ----
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(players)
	}
}
