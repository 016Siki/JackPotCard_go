package handlers

import (
	"api/internal/middleware"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// ChipResponse はユーザーの所持チップ数を返却するレスポンス構造体
type ChipResponse struct {
	SoloChip  int `json:"solo_chip"`  // ソロプレイ用チップ
	MultiChip int `json:"multi_chip"` // マルチプレイ用チップ
}

// GetChipDataHandler は認証済みユーザーのチップ数を返すHTTPハンドラ。
// JWT などの認証を middleware.GetUserID で確認し、DBからtipsテーブルを参照する。
func GetChipDataHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ---- 認証確認 ----
		userID := middleware.GetUserID(r)
		if userID == 0 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var soloChip, multiChip int
		err := db.QueryRow(`
            SELECT solo_tip_count, multi_tip_count
            FROM tips
            WHERE user_id = ?
        `, userID).Scan(&soloChip, &multiChip)
		// ---- エラーハンドリング ----
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Chip data not found", http.StatusNotFound)
			} else {
				log.Printf("[DB ERROR] %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
		// ---- レスポンス作成 ----
		response := ChipResponse{
			SoloChip:  soloChip,
			MultiChip: multiChip,
		}

		w.Header().Set("Content-Type", "application/json")
		// JSONでレスポンス返却
		json.NewEncoder(w).Encode(response)
	})
}
