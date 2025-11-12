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
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		roomCode := r.URL.Query().Get("code")
		if roomCode == "" {
			http.Error(w, "Missing room code", http.StatusBadRequest)
			return
		}

		room, err := models.GetRoomByCode(db, roomCode)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		users, err := models.GetUsersInRoom(db, room.ID)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		players := make([]PlayerInfo, 0, len(users))
		for _, u := range users {
			players = append(players, PlayerInfo{
				UserID:  u.UserID,
				Name:    u.UserName,
				IsReady: u.IsReady,
				IsHost:  u.IsHost,
			})
		}

		resp := RoomStatusResponse{
			Type:     "room_status",
			RoomCode: room.RoomCode,
			Players:  players,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
