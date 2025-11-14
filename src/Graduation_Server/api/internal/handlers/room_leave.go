package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"net/http"
)

// リクエスト
type LeaveRoomRequest struct {
	RoomCode string `json:"room_code"`
}

// レスポンス
type LeaveRoomResponse struct {
	Result         string `json:"result"` // "OK"
	RoomCode       string `json:"room_code"`
	HostChanged    bool   `json:"host_changed"`
	NewHostUserID  int64  `json:"new_host_user_id,omitempty"`
	RoomBecameZero bool   `json:"room_became_zero"`
}

func LeaveRoomHandler(db *sql.DB) http.HandlerFunc {
	// 認証 & リクエストチェック
	return func(w http.ResponseWriter, r *http.Request) {
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req LeaveRoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoomCode == "" {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// ルーム取得
		room, err := models.GetRoomByCode(db, req.RoomCode)
		if err != nil {
			http.Error(w, "Room not found", http.StatusNotFound)
			return
		}

		// 参加確認
		inRoom, err := models.IsUserInRoom(db, room.ID, userID)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		if !inRoom {
			http.Error(w, "Not in room", http.StatusForbidden)
			return
		}

		// Tx: 退出・ホスト交代・クローズ判定
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}
		defer func() { _ = tx.Rollback() }()

		// 退出
		affected, err := models.RemoveUserFromRoomTx(tx, room.ID, userID)
		if err != nil || affected == 0 {
			http.Error(w, "Leave failed", http.StatusInternalServerError)
			return
		}

		// 残人数チェック
		leftCount, err := models.CountUsersInRoomTx(tx, room.ID)
		if err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		resp := LeaveRoomResponse{
			Result:   "OK",
			RoomCode: room.RoomCode,
		}

		// 0人ならルームを閉じる（削除でもOK。ここでは status='closed' へ）
		if leftCount == 0 {
			if err := models.UpdateRoomStatusTx(tx, room.ID, "closed"); err != nil {
				http.Error(w, "DB error", http.StatusInternalServerError)
				return
			}
			resp.RoomBecameZero = true
		} else {
			// ホストが抜けたら交代
			if userID == room.OwnerID {
				newOwnerID, found, err := models.PickNextOwnerTx(tx, room.ID)
				if err != nil {
					http.Error(w, "DB error", http.StatusInternalServerError)
					return
				}
				if found {
					if err := models.SetRoomOwnerTx(tx, room.ID, newOwnerID); err != nil {
						http.Error(w, "DB error", http.StatusInternalServerError)
						return
					}
					resp.HostChanged = true
					resp.NewHostUserID = newOwnerID
				}
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "DB error", http.StatusInternalServerError)
			return
		}

		// このユーザーのWSを強制切断（ルーム側のコネクション表から掃除）
		closeUserConnections(req.RoomCode, userID)

		// 残メンバーへ最新状態を通知
		broadcastRoomStatus(req.RoomCode, db)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
