package handlers

import (
	"api/internal/middleware"
	"database/sql"
	"encoding/json"
	"net/http"
)

// UpdateTipRequest はチップ増減リクエストの構造体
// chip_diff: 増減させたいチップの差分値（+なら加算, -なら減算）
type UpdateTipRequest struct {
	NewChips int `json:"chip_diff"`
}

// UpdateSoloTipHandler はソロ用チップ数を更新するハンドラ。
// JWT 認証が必須で、リクエストで受け取った chip_diff を tips テーブルに反映する。
func UpdateSoloTipHandler(db *sql.DB) http.Handler {
	return middleware.JWTMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ---- ユーザーIDをJWTから取得 ----
		userIDFloat, ok := r.Context().Value(middleware.UserIDKey).(int64)
		if !ok {
			http.Error(w, "ユーザーIDが無効です", http.StatusUnauthorized)
			return
		}
		userID := int(userIDFloat)

		// ---- リクエストデコード ----
		// 例: {"chip_diff": -100}
		var req UpdateTipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "無効なリクエスト形式", http.StatusBadRequest)
			return
		}

		// ---- DB更新処理 ----
		// solo_tip_count を差分更新する
		_, err := db.Exec(
			"UPDATE tips SET solo_tip_count = solo_tip_count + ? WHERE user_id = ?",
			req.NewChips, userID,
		)
		if err != nil {
			http.Error(w, "チップ更新に失敗しました", http.StatusInternalServerError)
			return
		}

		// ---- 成功レスポンス ----
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("チップを更新しました"))
	}))
}
