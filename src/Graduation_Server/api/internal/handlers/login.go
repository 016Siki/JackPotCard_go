// ログインAPI
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// JWT署名用の秘密鍵
var jwtKey = []byte("your_secret_key")

// JWTに含めるクレーム構造体
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"name"`
	jwt.RegisteredClaims
}

// ログインリクエストの構造体
type LoginRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginHandler handles log inとJWTの発行
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	} else if strings.HasPrefix(ct, "application/x-www-form-urlencoded") {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form", http.StatusBadRequest)
			return
		}
		req.Name = r.FormValue("name")
		req.Password = r.FormValue("password")
	} else {
		http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
		return
	}

	if req.Name == "" || req.Password == "" {
		http.Error(w, "名前かパスワードがないよ", http.StatusBadRequest)
		return
	}

	// ユーザーのパスワードを取得
	var hashedPassword string
	var userID int64
	err := db.QueryRow("SELECT id, password FROM users WHERE name = ?", req.Name).Scan(&userID, &hashedPassword)
	if err == sql.ErrNoRows {
		http.Error(w, "ユーザーがありません", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// ハッシュが空の場合（AutoFlg = true で作成されたユーザー）は拒否
	if hashedPassword == "" {
		http.Error(w, "このアカウントではパスワードログインは無効です。", http.StatusUnauthorized)
		return
	}

	// パスワード照合
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		http.Error(w, "パスワードが正しくありません", http.StatusUnauthorized)
		return
	}

	// ログイン成功時JWT作成
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: req.Name,
		UserID:   userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	// JSONで返す
	resp := map[string]interface{}{
		"result":  "OK",
		"token":   tokenString,
		"message": "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
