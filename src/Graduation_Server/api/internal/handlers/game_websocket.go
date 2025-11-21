package handlers

import (
	"api/internal/middleware"
	"api/internal/models"
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

type PlayerMessage struct {
	PlayerID   string `json:"playerId"`
	PlayerName string `json:"playerName"`
	Chips      int    `json:"chips"`
	Action     string `json:"action"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Unityからの接続を許可
	},
}

// ブラックジャック用：書き込み排他制御
var bjWriteMu sync.Mutex

func writeJSONSafe(conn *websocket.Conn, v interface{}) error {
	bjWriteMu.Lock()
	defer bjWriteMu.Unlock()
	return conn.WriteJSON(v)
}
func init() {
	// ランダムの種
	rand.Seed(time.Now().UnixNano())
}

func BlackjackWebSocketHandle(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("WebSocket upgrade error:", err)
			return
		}
		defer conn.Close()

		log.Println("Blackjack WS connected")

		// ルームコード取得（/api/ws/blackjackwebsocket/{room_code}）
		roomCode := mux.Vars(r)["room_code"]
		log.Println("[BJWS] roomCode =", roomCode)

		// ユーザーID取得（JWTから）
		userID := middleware.GetUserID(r)
		if userID == 0 {
			log.Println("Unauthorized")
			return
		}
		log.Println("[BJWS] userID =", userID)

		// ルーム情報取得
		room, err := models.GetRoomByCode(db, roomCode)
		if err != nil {
			log.Println("Room not found:", err)
			return
		}

		// DBから現在のプレイヤー一覧取得
		users, err := models.GetUsersInRoom(db, room.ID)
		if err != nil {
			log.Println("GetUsersInRoom failed:", err)
			return
		}
		if len(users) == 0 {
			log.Println("[BJWS] no users in room")
			return
		}

		// ===== ブラックジャック用メモリ状態初期化 =====
		bjMu.Lock()

		state, ok := bjRoomStates[roomCode]
		if !ok {
			state = &BJRoomState{
				Players: make(map[int64]*BJBetPlayerState),
			}
			bjRoomStates[roomCode] = state
		}

		// プレイヤー状態を BJRoomState に登録
		for _, u := range users {
			if _, exists := state.Players[u.UserID]; !exists {
				state.Players[u.UserID] = &BJBetPlayerState{
					UserID:     u.UserID,
					Name:       u.UserName,
					Bet:        0,
					Confirmed:  false,
					TotalChips: 1000,
				}
			}
		}

		// ★ ここで DealerID を 1 回だけ決定 or 既存のものを使う
		dealerID := EnsureDealerAssigned(state)

		// 接続管理
		if bjRoomConns[roomCode] == nil {
			bjRoomConns[roomCode] = make(map[*websocket.Conn]int64)
		}
		bjRoomConns[roomCode][conn] = userID

		bjMu.Unlock()

		// ===== PlayerInfo配列を作成 =====
		players := make([]PlayerInfo, 0, len(users))
		for _, u := range users {
			players = append(players, PlayerInfo{
				UserID:   u.UserID,
				Name:     u.UserName,
				IsReady:  u.IsReady,
				IsHost:   u.IsHost,
				IsDealer: (u.UserID == dealerID), // ★ ここだけでOK
			})
		}

		// ===== 最初にプレイヤー並び情報を送信 =====
		if err := SendPlayerOrder(conn, players); err != nil {
			log.Println("player_order send error:", err)
		} else {
			log.Printf("[BJWS] player_order sent (dealerID=%d)\n", dealerID)
		}

		// 入室直後に現在のベット状態を送る
		broadcastBetState(roomCode)

		// ゲーム開始タイミングで 15秒タイマー開始（暫定：接続時）
		startActionTimer(conn, 15.0)

		// ===== 受信ループ =====
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				log.Println("Blackjack WS closed:", err)
				break
			}

			log.Println("Blackjack Received:", string(raw))

			var cmd BetCommand
			if err := json.Unmarshal(raw, &cmd); err != nil {
				log.Println("BetCommand json error:", err)
				continue
			}

			if cmd.Type != "bet_update" {
				log.Println("Unknown blackjack ws message type:", cmd.Type)
				continue
			}

			// ==== ベット更新処理 ====
			bjMu.Lock()
			st, ok := bjRoomStates[roomCode]
			if !ok {
				bjMu.Unlock()
				continue
			}
			p, ok := st.Players[userID]
			if !ok {
				bjMu.Unlock()
				continue
			}

			// ディーラーはベット不可（サーバー側でも念のため弾く）
			if st.DealerID == userID {
				log.Println("[BJWS] dealer tried to bet, ignored")
				bjMu.Unlock()
				continue
			}
			if cmd.Bet < 0 {
				log.Println("[BJWS] negative bet ignored:", cmd.Bet)
				bjMu.Unlock()
				continue
			}
			oldBet := p.Bet
			newBet := cmd.Bet
			delta := newBet - oldBet // 例: old=100 new=300 → delta=+200 (追加で200)

			if delta > 0 {
				// 追加で賭ける分のチップが足りるかチェック
				if p.TotalChips < delta {
					log.Printf("[BJWS] user %d not enough chips. have=%d, need=%d\n",
						p.UserID, p.TotalChips, delta)
					bjMu.Unlock()
					continue
				}
				p.TotalChips -= delta
			} else if delta < 0 {
				// ベット額を減らした場合は、差分だけチップを戻す（必要なら）
				// もし「一度確定したら減らせない」仕様にするなら、ここは無視してもOK
				p.TotalChips -= delta // delta はマイナスなので実質 +abs(delta)
			}

			p.Bet = cmd.Bet
			p.Confirmed = cmd.Confirm

			bjMu.Unlock()

			// 全員分の最新状態を broadcast
			broadcastBetState(roomCode)
		}

		// ===== 接続終了処理 =====
		bjMu.Lock()
		if m, ok := bjRoomConns[roomCode]; ok {
			delete(m, conn)
			if len(m) == 0 {
				delete(bjRoomConns, roomCode)
				// 必要なら部屋の状態もクリア
				// delete(bjRoomStates, roomCode)
			}
		}
		bjMu.Unlock()
	}
}
