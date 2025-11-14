package models

import "database/sql"

// room_users から room_id ごとの人数を数える
func CountUsersInRoomTx(tx *sql.Tx, roomID int64) (int, error) {
	var cnt int
	err := tx.QueryRow(`SELECT COUNT(*) FROM room_users WHERE room_id = ?`, roomID).Scan(&cnt)
	return cnt, err
}

// LeaveRoomHandler で、残り 0 人になったとき status='closed' にするのに使用。
func UpdateRoomStatusTx(tx *sql.Tx, roomID int64, status string) error {
	_, err := tx.Exec(`UPDATE rooms SET status = ? WHERE id = ?`, status, roomID)
	return err
}

// MIN(ru.id) のユーザーを次のオーナーに採用（別基準が良ければ変更してOK）
func PickNextOwnerTx(tx *sql.Tx, roomID int64) (userID int64, found bool, err error) {
	err = tx.QueryRow(`
		SELECT ru.user_id
		  FROM room_users ru
		 WHERE ru.room_id = ?
		 ORDER BY ru.id ASC
		 LIMIT 1
	`, roomID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return userID, true, nil
}

// rooms.owner_id = newOwnerID に更新するだけの関数。
func SetRoomOwnerTx(tx *sql.Tx, roomID, newOwnerID int64) error {
	_, err := tx.Exec(`UPDATE rooms SET owner_id = ? WHERE id = ?`, newOwnerID, roomID)
	return err
}

// 退出（行削除）。戻り値は削除件数。
func RemoveUserFromRoomTx(tx *sql.Tx, roomID, userID int64) (int64, error) {
	res, err := tx.Exec(`DELETE FROM room_users WHERE room_id = ? AND user_id = ?`, roomID, userID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
