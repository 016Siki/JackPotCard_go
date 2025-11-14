package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// リクエスト
type StartRequest struct {
	RoomCode string `json:"room_code"`
}

// レスポンス
type StartResponse struct {
	Result string `json:"result"`
	GameID int64  `json:"game_id"`
}

func StartRoomHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 認証 & リクエストチェック
		userID := middleware.GetUserID(r)
		log.Printf("[StartRoom] Request userID = %d\n", userID)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		var req StartRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoomCode == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		// ルーム取得
		room, err := models.GetRoomByCode(db, req.RoomCode)
		if err != nil {
			http.Error(w, "Room not found: "+err.Error(), http.StatusNotFound)
			return
		}

		log.Printf("[StartRoom] room.ID=%d, room.OwnerID=%d, userID=%d\n",
			room.ID, room.OwnerID, userID)

		if room.OwnerID != userID {
			log.Printf("[StartRoom] forbidden: not host (userID=%d, ownerID=%d)\n", userID, room.OwnerID)
			http.Error(w, "Only host can start", http.StatusForbidden)
			return
		}
		// 「ホストかどうか」の判定
		// isHost, err := models.IsUserHostInRoom(db, room.ID, userID)
		// if err != nil {
		// 	http.Error(w, "host check failed: "+err.Error(), http.StatusInternalServerError)
		// 	return
		// }
		// if !isHost {
		// 	http.Error(w, "Only host can start", http.StatusForbidden)
		// 	return
		// }
		// 全員 Ready かチェック
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
		// 状態チェック
		if room.Status == "playing" {
			http.Error(w, "Already playing", http.StatusConflict)
			return
		}
		// トランザクションで Game 作成 & Room 状態更新
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
		// WebSocket へ「ゲーム開始」を通知 & レスポンス返却
		go broadcastStartGame(req.RoomCode, gameID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(StartResponse{Result: "OK", GameID: gameID})
	}
}
