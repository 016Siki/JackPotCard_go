package models

import "database/sql"

func AddUserToRoom(db *sql.DB, roomID, userID int64) error {
	_, err := db.Exec(`INSERT INTO room_users (room_id, user_id, is_ready) VALUES (?, ?, false)`, roomID, userID)
	return err
}

func CountUsersInRoom(db *sql.DB, roomID int64) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM room_users WHERE room_id = ?`, roomID).Scan(&count)
	return count, err
}

func UpdateUserReady(db *sql.DB, roomID, userID int64, isReady bool) error {
	_, err := db.Exec(`UPDATE room_users SET is_ready = ? WHERE room_id = ? AND user_id = ?`, isReady, roomID, userID)
	return err
}

func CountReadyUsers(db *sql.DB, roomID int64) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM room_users WHERE room_id = ? AND is_ready = true`, roomID).Scan(&count)
	return count, err
}
func AddUserToRoomAsHost(tx *sql.Tx, roomID, userID int64, isReady bool) error {
	// (room_id, user_id) に UNIQUE がある前提（無い場合は下のDDLを参照）
	_, err := tx.Exec(`
		INSERT INTO room_users (room_id, user_id, is_ready)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			is_ready = VALUES(is_ready)
	`, roomID, userID, isReady)
	return err
}

// ユーザーがそのルームに参加済みかどうか（room_users の存在チェック）
func IsUserInRoom(db *sql.DB, roomID, userID int64) (bool, error) {
	var dummy int
	err := db.QueryRow(`
		SELECT 1 FROM room_users WHERE room_id = ? AND user_id = ? LIMIT 1
	`, roomID, userID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// 対象ユーザーがホストかどうか（rooms.owner_id で判定）
func IsUserHostInRoom(db *sql.DB, roomID, userID int64) (bool, error) {
	var dummy int
	err := db.QueryRow(`
		SELECT 1 FROM rooms WHERE id = ? AND owner_id = ? LIMIT 1
	`, roomID, userID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
