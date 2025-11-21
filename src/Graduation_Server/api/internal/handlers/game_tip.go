package handlers

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// プレイヤーごとのベット状態
type BJBetPlayerState struct {
	UserID     int64  `json:"user_id"`
	Name       string `json:"name"`
	Bet        int    `json:"bet"`
	Confirmed  bool   `json:"confirmed"`
	TotalChips int    `json:"total_chips"`
}

// ルームごとのブラックジャック状態
type BJRoomState struct {
	Players  map[int64]*BJBetPlayerState // userID -> state
	DealerID int64
}

// ルームコードごとの状態・接続
var (
	bjRoomStates = make(map[string]*BJRoomState)              // roomCode -> state
	bjRoomConns  = make(map[string]map[*websocket.Conn]int64) // roomCode -> (conn -> userID)
	bjMu         sync.Mutex
)

// クライアント→サーバー：ベット更新コマンド
type BetCommand struct {
	Type    string `json:"type"`    // "bet_update"
	Bet     int    `json:"bet"`     // 賭けチップ
	Confirm bool   `json:"confirm"` // 決定ボタン押したか
}

// サーバー→クライアント：ベット状態ブロードキャスト
type BetStateBroadcast struct {
	Type         string             `json:"type"` // "bet_state"
	RoomCode     string             `json:"room_code"`
	Players      []BJBetPlayerState `json:"players"`
	AllConfirmed bool               `json:"all_confirmed"` // 全員決定済みか
}

// 全員の状態をそのルームの全WS接続へブロードキャスト
func broadcastBetState(roomCode string) {
	bjMu.Lock()
	state, ok := bjRoomStates[roomCode]
	if !ok {
		bjMu.Unlock()
		return
	}
	conns := bjRoomConns[roomCode]
	// スナップショット作成
	var players []BJBetPlayerState
	allConfirmed := true
	for _, p := range state.Players {
		players = append(players, *p)
		// ★ ディーラーは allConfirmed 判定から除外
		if p.UserID == state.DealerID {
			continue
		}
		if !p.Confirmed {
			allConfirmed = false
		}
	}
	bjMu.Unlock()

	res := BetStateBroadcast{
		Type:         "bet_state",
		RoomCode:     roomCode,
		Players:      players,
		AllConfirmed: allConfirmed,
	}

	var toDelete []*websocket.Conn

	for c := range conns {
		if err := writeJSONSafe(c, res); err != nil {
			log.Println("bet_state send error:", err)
			toDelete = append(toDelete, c)
		}
	}

	if len(toDelete) > 0 {
		bjMu.Lock()
		defer bjMu.Unlock()
		for _, c := range toDelete {
			delete(bjRoomConns[roomCode], c)
			_ = c.Close()
		}
	}
}
