package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

type JoinRoomRequest struct {
	RoomCode string `json:"room_code"`
}

type JoinRoomResponse struct {
	Result   string `json:"result"`
	RoomID   int64  `json:"room_id"`
	RoomCode string `json:"room_code"`
	UserID   int64  `json:"user_id"`
	GameName string `json:"game_name"`
}

func JoinRoomHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req JoinRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoomCode == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		room, err := models.GetRoomByCode(db, req.RoomCode)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}
		// 待機中以外は参加不可
		if room.Status != "waiting" {
			http.Error(w, "Room not accepting joins", http.StatusForbidden)
			return
		}

		// すでに参加していないか
		inRoom, err := models.IsUserInRoom(db, room.ID, userID)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		if inRoom {
			// 既参加ならOKで返す（重複insertを避ける）
			var gameName string
			if err := db.QueryRow(`
				SELECT gt.name
				  FROM game_types gt
				  JOIN rooms r ON r.game_type_id = gt.id
				 WHERE r.id = ?
			`, room.ID).Scan(&gameName); err != nil {
				http.Error(w, "Lookup error", http.StatusInternalServerError)
				return
			}
			resp := JoinRoomResponse{
				Result:   "OK",
				RoomID:   room.ID,
				RoomCode: room.RoomCode,
				UserID:   userID,
				GameName: gameName,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		count, err := models.CountUsersInRoom(db, room.ID)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		if count >= room.MaxPlayers {
			http.Error(w, "Room full", http.StatusForbidden)
			return
		}

		if err := models.AddUserToRoom(db, room.ID, userID); err != nil {
			log.Printf("AddUserToRoom error: %v", err)
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		var gameName string
		if err := db.QueryRow(`
			SELECT gt.name
			  FROM game_types gt
			  JOIN rooms r ON r.game_type_id = gt.id
			 WHERE r.id = ?
		`, room.ID).Scan(&gameName); err != nil {
			log.Printf("lookup game_name failed: %v", err)
			http.Error(w, "Lookup error", http.StatusInternalServerError)
			return
		}

		resp := JoinRoomResponse{
			Result:   "OK",
			RoomID:   room.ID,
			RoomCode: room.RoomCode,
			UserID:   userID,
			GameName: gameName,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)

		// 参加後はWS側にも反映
		broadcastRoomStatus(room.RoomCode, db)
	}
}
