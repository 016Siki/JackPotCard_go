package handlers

import (
	"encoding/json"
	"net/http"

	"api/internal/middleware"
)

// UpdateUserSettingsRequest はユーザーの音量設定を更新するリクエスト
// bgm_volume: BGM音量 (0.0〜1.0)
// se_volume:  効果音(SE)音量 (0.0〜1.0)
type UpdateUserSettingsRequest struct {
	BgmVolume float64 `json:"bgm_volume"`
	SeVolume  float64 `json:"se_volume"`
}

// UpdateUserSettingsResponse は更新結果のレスポンス
// result: "OK" 固定
type UpdateUserSettingsResponse struct {
	Result string `json:"result"`
}

// UpdateUserSettingsHandler はユーザーの音量設定を更新するハンドラ。
// JWT認証で userID を確認し、settings テーブルを更新する。
func UpdateUserSettingsHandler(w http.ResponseWriter, r *http.Request) {
	// ---- 認証確認 ----
	userID := middleware.GetUserID(r)
	if userID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// ---- リクエストデコード ----
	var req UpdateUserSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// ---- トランザクション開始 ----
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "DB transaction error", http.StatusInternalServerError)
		return
	}
	// エラー時は自動でRollbackされる
	defer tx.Rollback()

	// ---- settings 更新処理 ----
	_, err = tx.Exec(`
		UPDATE settings
		SET bgm_volume = ?, se_volume = ?
		WHERE user_id = ?`,
		req.BgmVolume, req.SeVolume, userID)
	if err != nil {
		http.Error(w, "Failed to update volume settings", http.StatusInternalServerError)
		return
	}

	// ---- コミット ----
	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit DB transaction", http.StatusInternalServerError)
		return
	}

	// ---- 成功レスポンス返却 ----
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(UpdateUserSettingsResponse{Result: "OK"})
}
