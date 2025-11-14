package models

import (
	"database/sql"
)

// 型定義
type PlayerInfo struct {
	Name    string `json:"name"`
	IsReady bool   `json:"is_ready"`
	UserID  int64  `json:"user_id"`
	IsHost  bool   `json:"is_host"`
}

func GetPlayersByRoomID(db *sql.DB, roomID int64) ([]PlayerInfo, error) {
	rows, err := db.Query(`
        SELECT u.name, ru.is_ready, ru.user_id, 
               CASE WHEN ru.user_id = r.owner_id THEN 1 ELSE 0 END AS is_host
        FROM room_users ru
        INNER JOIN users u ON ru.user_id = u.id
        INNER JOIN rooms r ON ru.room_id = r.id
        WHERE ru.room_id = ?`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []PlayerInfo
	for rows.Next() {
		var p PlayerInfo
		var isHostInt int
		if err := rows.Scan(&p.Name, &p.IsReady, &p.UserID, &isHostInt); err != nil {
			return nil, err
		}
		p.IsHost = isHostInt == 1
		players = append(players, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return players, nil
}

func GetGameNameByTypeID(db *sql.DB, gameTypeID int) (string, error) {
	var name string
	err := db.QueryRow("SELECT name FROM game_types WHERE id = ?", gameTypeID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "不明なゲーム", nil
		}
		return "", err
	}
	return name, nil
}
