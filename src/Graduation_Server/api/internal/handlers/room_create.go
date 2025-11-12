package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"api/internal/utils"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type CreateRoomRequest struct {
	GameTypeID int `json:"game_type_id"`
	MaxPlayers int `json:"max_players"`
}

type CreateRoomResponse struct {
	Result   string `json:"result"`
	RoomCode string `json:"room_code"`
	UserID   int64  `json:"user_id"`
	GameName string `json:"game_name"`
}

func CreateRoomHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req CreateRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if req.MaxPlayers <= 0 {
			req.MaxPlayers = 4 // デフォルト（任意）
		}

		roomCode := utils.GenerateRoomCode()
		now := time.Now()

		var gameName string
		if err := db.QueryRow("SELECT name FROM game_types WHERE id = ?", req.GameTypeID).Scan(&gameName); err != nil {
			log.Printf("[DB ERROR] game_types lookup failed: %v", err)
			http.Error(w, "Invalid game_type_id", http.StatusBadRequest)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer func() { _ = tx.Rollback() }()

		res, err := tx.Exec(
			"INSERT INTO rooms (room_code, game_type_id, max_players, owner_id, status, created_at) VALUES (?, ?, ?, ?, ?, ?)",
			roomCode, req.GameTypeID, req.MaxPlayers, userID, "waiting", now,
		)
		if err != nil {
			log.Printf("[DB ERROR] rooms insert failed: %v", err)
			http.Error(w, "DB Insert Error", http.StatusInternalServerError)
			return
		}
		roomID, err := res.LastInsertId()
		if err != nil {
			http.Error(w, "DB Insert Error", http.StatusInternalServerError)
			return
		}

		// 参加情報（ホスト＆未準備）で登録
		if err := models.AddUserToRoomAsHost(tx, roomID, userID, false /*is_ready*/); err != nil {
			log.Printf("[DB ERROR] room_users insert failed: %v", err)
			http.Error(w, "DB Insert Error", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		resp := CreateRoomResponse{
			Result:   "OK",
			RoomCode: roomCode,
			UserID:   userID,
			GameName: gameName,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
