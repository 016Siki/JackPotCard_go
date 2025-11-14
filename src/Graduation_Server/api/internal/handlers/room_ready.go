package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

// ReadyResponse は「準備完了」API成功時に返すレスポンス。
type ReadyResponse struct {
	Result string `json:"result"`
}

// ReadyHandler は「ルーム内でユーザーが準備完了状態になった」ことを更新するハンドラ。
func ReadyHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ユーザーID取得
		userID := middleware.GetUserID(r)
		var req ReadyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		room, err := models.GetRoomByCode(db, req.RoomCode)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		if err := models.UpdateUserReady(db, room.ID, userID, req.IsReady); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(ReadyResponse{Result: "OK"})
	}
}
