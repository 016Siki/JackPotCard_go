package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

func RoomStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 認証チェック
		// 未認証は401
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		// ルームコード取得
		roomCode := r.URL.Query().Get("code")
		if roomCode == "" {
			http.Error(w, "Missing room code", http.StatusBadRequest)
			return
		}
		// DB からルーム情報を取得
		room, err := models.GetRoomByCode(db, roomCode)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		// ルーム内の参加メンバー一覧を取得
		users, err := models.GetUsersInRoom(db, room.ID)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		// プレイヤー情報をレスポンス用に変換
		// Unity に渡すデータ構造に変換
		// Host かどうか、Ready 状態も含む
		players := make([]PlayerInfo, 0, len(users))
		for _, u := range users {
			players = append(players, PlayerInfo{
				UserID:  u.UserID,
				Name:    u.UserName,
				IsReady: u.IsReady,
				IsHost:  u.IsHost,
			})
		}
		// レスポンスを返す
		resp := RoomStatusResponse{
			Type:     "room_status",
			RoomCode: room.RoomCode,
			Players:  players,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
