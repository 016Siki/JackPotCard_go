// internal/handlers/blackjack_dealer.go
package handlers

import (
	"math/rand"
	"time"
)

// このファイル内だけの初期化（乱数シード）
func init() {
	rand.Seed(time.Now().UnixNano())
}

// players スライスの中からランダムに1人をディーラーにして IsDealer=true を付ける。
// すでに IsDealer がついていても一旦クリアしてから再度付け直す。
// 戻り値: 更新済み players, dealerIndex (ディーラーの index / 見つからなければ -1)
func EnsureDealerAssigned(state *BJRoomState) int64 {
	if state.DealerID != 0 {
		return state.DealerID
	}

	ids := make([]int64, 0, len(state.Players))
	for id := range state.Players {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return 0
	}

	dealerID := ids[rand.Intn(len(ids))]
	state.DealerID = dealerID
	if p, ok := state.Players[dealerID]; ok {
		p.Bet = 0
		p.Confirmed = true
	}
	return dealerID
}
