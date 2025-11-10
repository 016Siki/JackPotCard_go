package models

import (
	"database/sql"
)

type PlayerGameInfo struct {
	UserID   int64  `json:"user_id"`
	UserName string `json:"user_name"`
	Chip     int64  `json:"chip"`
	Bet      int64  `json:"bet"`
	IsDealer bool   `json:"is_dealer"`
}

// room_idに紐づくゲーム用プレイヤー情報を取得
func GetPlayersForGame(db *sql.DB, roomID int64) ([]PlayerGameInfo, error) {
	rows, err := db.Query(`
        SELECT u.id, u.name, p.chip, p.bet, p.is_dealer
        FROM players p
        JOIN users u ON p.user_id = u.id
        WHERE p.room_id = ?
    `, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []PlayerGameInfo
	for rows.Next() {
		var p PlayerGameInfo
		if err := rows.Scan(&p.UserID, &p.UserName, &p.Chip, &p.Bet, &p.IsDealer); err != nil {
			return nil, err
		}
		players = append(players, p)
	}
	return players, nil
}
