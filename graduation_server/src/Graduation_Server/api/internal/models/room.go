package models

import "database/sql"

type Room struct {
	ID         int64
	RoomCode   string
	GameTypeID int
	Status     string
	MaxPlayers int
	CreatedAt  string
	OwnerID    int64
}

type RoomUser struct {
	ID       int64  `json:"id"`      // ← これは room_users.id として残す（任意）
	UserID   int64  `json:"user_id"` // ← これを string → int64 にし、users.id を格納
	RoomID   int64  `json:"room_id"`
	UserName string `json:"user_name"`
	IsReady  bool   `json:"is_ready"`
	IsHost   bool   `json:"is_host"`
}

func CreateRoom(db *sql.DB, roomCode string, gameTypeID, maxPlayers int) (int64, error) {
	res, err := db.Exec(`INSERT INTO rooms (room_code, game_type_id, status, max_players) VALUES (?, ?, 'waiting', ?)`,
		roomCode, gameTypeID, maxPlayers)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func GetRoomByCode(db *sql.DB, roomCode string) (*Room, error) {
	var r Room
	err := db.QueryRow(`
		SELECT id, room_code, game_type_id, status, max_players, created_at, owner_id
		  FROM rooms
		 WHERE room_code = ?`,
		roomCode,
	).Scan(&r.ID, &r.RoomCode, &r.GameTypeID, &r.Status, &r.MaxPlayers, &r.CreatedAt, &r.OwnerID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func UpdateRoomStatus(db *sql.DB, roomID int64, status string) error {
	_, err := db.Exec(`UPDATE rooms SET status = ? WHERE id = ?`, status, roomID)
	return err
}

type CreateRoomRequest struct {
	GameTypeID int `json:"game_type_id"`
	MaxPlayers int `json:"max_players"`
}

func GetUsersInRoom(db *sql.DB, roomID int64) ([]RoomUser, error) {
	rows, err := db.Query(`
SELECT 
    ru.id,                  -- room_users.id（内部ID）
    ru.room_id,
    u.id AS user_id,        -- ← ここを追加（users.id）
    u.name AS user_name,
    ru.is_ready,
    CASE WHEN ru.user_id = r.owner_id THEN 1 ELSE 0 END AS is_host
FROM room_users ru
INNER JOIN users u ON ru.user_id = u.id
INNER JOIN rooms r ON ru.room_id = r.id
WHERE ru.room_id = ?
ORDER BY is_host DESC, ru.id ASC

    `, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []RoomUser
	for rows.Next() {
		var u RoomUser
		var isHostInt int
		if err := rows.Scan(&u.ID, &u.RoomID, &u.UserID, &u.UserName, &u.IsReady, &isHostInt); err != nil {
			return nil, err
		}
		u.IsHost = isHostInt == 1
		users = append(users, u)
	}
	return users, nil
}
