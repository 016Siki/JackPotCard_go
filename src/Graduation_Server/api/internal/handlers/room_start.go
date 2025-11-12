package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

type StartRequest struct {
	RoomCode string `json:"room_code"`
}
type StartResponse struct {
	Result string `json:"result"`
	GameID int64  `json:"game_id"`
}

func StartRoomHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var req StartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoomCode == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		room, err := models.GetRoomByCode(db, req.RoomCode)
		if err != nil {
			http.Error(w, "Room not found: "+err.Error(), http.StatusNotFound)
			return
		}

		isHost, err := models.IsUserHostInRoom(db, room.ID, userID)
		if err != nil {
			http.Error(w, "host check failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if !isHost {
			http.Error(w, "Only host can start", http.StatusForbidden)
			return
		}

		userCount, err := models.CountUsersInRoom(db, room.ID)
		if err != nil {
			http.Error(w, "count users failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		readyCount, err := models.CountReadyUsers(db, room.ID)
		if err != nil {
			http.Error(w, "count ready failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if userCount == 0 || userCount != readyCount {
			http.Error(w, "Not all users are ready", http.StatusForbidden)
			return
		}
		if room.Status == "playing" {
			http.Error(w, "Already playing", http.StatusConflict)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "tx begin failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = tx.Rollback() }()

		res, err := tx.Exec(`
			INSERT INTO games (mode_id, type_id, created_at, updated_at)
			VALUES (?, ?, NOW(), NOW())`,
			1, room.GameTypeID,
		)
		if err != nil {
			http.Error(w, "insert games failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		gameID, err := res.LastInsertId()
		if err != nil {
			http.Error(w, "lastInsertId failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := tx.Exec(`UPDATE rooms SET status='playing' WHERE id=?`, room.ID); err != nil {
			http.Error(w, "update room status failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, "tx commit failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		go broadcastStartGame(req.RoomCode, gameID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(StartResponse{Result: "OK", GameID: gameID})
	}
}
