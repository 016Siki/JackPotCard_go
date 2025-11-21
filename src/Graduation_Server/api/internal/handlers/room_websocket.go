package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var roomConnections = make(map[string]map[*websocket.Conn]int64)
var roomConnMu sync.Mutex

// WebSocket メッセージ構造
type ReadyRequest struct {
	RoomCode string `json:"room_code"`
	IsReady  bool   `json:"is_ready"`
}

type PlayerInfo struct {
	UserID   int64  `json:"user_id"`
	Name     string `json:"name"`
	IsReady  bool   `json:"is_ready"`
	IsHost   bool   `json:"is_host"`
	IsDealer bool   `json:"is_dealer"`
}

type RoomStatusResponse struct {
	Type       string       `json:"type"` // "room_status"
	RoomCode   string       `json:"room_code"`
	Players    []PlayerInfo `json:"players"`
	MaxPlayers int          `json:"max_players"`
}

type AllReadyResponse struct {
	Type     string `json:"type"` // "all_ready"
	RoomCode string `json:"room_code"`
	AllReady bool   `json:"all_ready"`
}
type StartGameBroadcast struct {
	Type     string `json:"type"` // "start_game"
	RoomCode string `json:"room_code"`
	GameID   int64  `json:"game_id"`
}

func GameRoomWebSocketHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// オリジン許可（必要に応じてmiddleware.Upgraderで設定）
		conn, err := middleware.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket Upgrade Error:", err)
			return
		}
		log.Println("WS connected")
		defer func() {
			log.Println("WS closing")
			conn.Close()
		}()

		roomCode := mux.Vars(r)["room_code"]
		userID := middleware.GetUserID(r)
		if userID == 0 {
			log.Println("Unauthorized WebSocket access (userID = 0)")
			return
		}

		// ルーム存在チェック
		room, err := models.GetRoomByCode(db, roomCode)
		if err != nil {
			log.Println("Room not found:", roomCode, err)
			return
		}
		// 所属チェック（未所属なら弾く）
		inRoom, err := models.IsUserInRoom(db, room.ID, userID)
		if err != nil || !inRoom {
			log.Printf("user %d is not in room %s\n", userID, roomCode)
			return
		}

		// 受信制限 & 死活監視
		conn.SetReadLimit(50 << 20) // 20MB
		_ = conn.SetReadDeadline(time.Now().Add(180 * time.Second))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(180 * time.Second))
		})

		// Ping送信用ゴルーチン
		stopPing := make(chan struct{})
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second)); err != nil {
						return
					}
				case <-stopPing:
					return
				}
			}
		}()

		// 接続登録
		roomConnMu.Lock()
		if roomConnections[roomCode] == nil {
			roomConnections[roomCode] = make(map[*websocket.Conn]int64)
		}
		roomConnections[roomCode][conn] = userID
		roomConnMu.Unlock()

		defer func() {
			close(stopPing)
			// 切断 → Readyをfalseに（任意）
			_ = models.UpdateUserReady(db, room.ID, userID, false)

			roomConnMu.Lock()
			delete(roomConnections[roomCode], conn)
			roomConnMu.Unlock()

			log.Printf("WebSocket disconnected: roomCode=%s, userID=%d\n", roomCode, userID)
			// 切断後に最新状態を通知
			broadcastRoomStatus(roomCode, db)
		}()

		// 接続直後の同期
		broadcastRoomStatus(roomCode, db)

		for {
			msgType, raw, err := conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket Read Error:", err)
				break
			}
			if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
				continue
			}

			var readyReq ReadyRequest
			if err := json.Unmarshal(raw, &readyReq); err != nil {
				log.Println("Invalid message format:", err)
				continue
			}
			// room_codeの偽装防止：URLのroomCode固定で進めたいならここで上書きする
			if readyReq.RoomCode != roomCode {
				readyReq.RoomCode = roomCode
			}

			room, err := models.GetRoomByCode(db, readyReq.RoomCode)
			if err != nil {
				log.Println("GetRoomByCode error:", err)
				continue
			}

			if err := models.UpdateUserReady(db, room.ID, userID, readyReq.IsReady); err != nil {
				log.Println("UpdateUserReady error:", err)
				continue
			}

			broadcastRoomStatus(readyReq.RoomCode, db)
		}
	}
}

func broadcastRoomStatus(roomCode string, db *sql.DB) {
	roomConnMu.Lock()
	connMap := roomConnections[roomCode]
	if len(connMap) == 0 {
		roomConnMu.Unlock()
		return
	}
	type item struct {
		c   *websocket.Conn
		uid int64
	}
	snapshot := make([]item, 0, len(connMap))
	for c, uid := range connMap {
		snapshot = append(snapshot, item{c: c, uid: uid})
	}
	roomConnMu.Unlock()

	room, err := models.GetRoomByCode(db, roomCode)
	if err != nil {
		log.Println("GetRoomByCode failed:", err)
		return
	}

	users, err := models.GetUsersInRoom(db, room.ID)
	if err != nil {
		log.Println("GetUsersInRoom failed:", err)
		return
	}

	players := make([]PlayerInfo, 0, len(users))
	allReady := true
	var hostUserID int64

	for _, u := range users {
		players = append(players, PlayerInfo{
			UserID:  u.UserID,
			Name:    u.UserName,
			IsReady: u.IsReady,
			IsHost:  u.IsHost,
		})
		if !u.IsReady {
			allReady = false
		}
		if u.IsHost {
			hostUserID = u.UserID
		}
	}

	statusMsg := RoomStatusResponse{
		Type:       "room_status",
		RoomCode:   roomCode,
		Players:    players,
		MaxPlayers: room.MaxPlayers,
	}

	var toDelete []*websocket.Conn
	for _, it := range snapshot {
		if err := it.c.WriteJSON(statusMsg); err != nil {
			log.Println("Broadcast error:", err)
			toDelete = append(toDelete, it.c)
			continue
		}
		// 全員Readyならホストにだけall_readyを送る
		if allReady && it.uid == hostUserID {
			readyMsg := AllReadyResponse{
				Type:     "all_ready",
				RoomCode: roomCode,
				AllReady: true,
			}
			if err := it.c.WriteJSON(readyMsg); err != nil {
				log.Println("AllReady broadcast error:", err)
				toDelete = append(toDelete, it.c)
			}
		}
	}

	if len(toDelete) > 0 {
		roomConnMu.Lock()
		for _, c := range toDelete {
			delete(roomConnections[roomCode], c)
			_ = c.Close()
		}
		roomConnMu.Unlock()
	}
}

// 退出/キック時に、該当ユーザーのWSコネクションを閉じる
func closeUserConnections(roomCode string, userID int64) {
	roomConnMu.Lock()
	defer roomConnMu.Unlock()

	connMap := roomConnections[roomCode]
	if connMap == nil {
		return
	}

	for c, uid := range connMap {
		if uid == userID {
			delete(connMap, c)
			_ = c.Close()
		}
	}
	// 空ならmapごと掃除（任意）
	if len(connMap) == 0 {
		delete(roomConnections, roomCode)
	}
}
func broadcastStartGame(roomCode string, gameID int64) {
	roomConnMu.Lock()
	connMap := roomConnections[roomCode]
	if len(connMap) == 0 {
		roomConnMu.Unlock()
		return
	}
	type item struct {
		c   *websocket.Conn
		uid int64
	}
	snapshot := make([]item, 0, len(connMap))
	for c, uid := range connMap {
		snapshot = append(snapshot, item{c: c, uid: uid})
	}
	roomConnMu.Unlock()

	msg := StartGameBroadcast{
		Type:     "start_game",
		RoomCode: roomCode,
		GameID:   gameID,
	}

	var toDelete []*websocket.Conn
	for _, it := range snapshot {
		// （任意）送信詰まり防止の締切
		_ = it.c.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := it.c.WriteJSON(msg); err != nil {
			// 送れない接続は掃除
			toDelete = append(toDelete, it.c)
		}
		// 締切は都度リセットしておくと無難
		_ = it.c.SetWriteDeadline(time.Time{})
	}

	if len(toDelete) > 0 {
		roomConnMu.Lock()
		for _, c := range toDelete {
			delete(roomConnections[roomCode], c)
			_ = c.Close()
		}
		roomConnMu.Unlock()
	}
}
