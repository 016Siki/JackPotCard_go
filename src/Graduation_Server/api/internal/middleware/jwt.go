package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

// コンテキストに格納するキーの型（衝突防止）
type contextKey string

const UserIDKey = contextKey("userID")

// JWTの署名鍵（環境変数などで管理推奨）
var jwtSecret = []byte("your_secret_key")

// JWTミドルウェア
func JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tokenStr string

		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Authorizationヘッダーなければクエリパラメータtokenを取得
			tokenStr = r.URL.Query().Get("token")
		}

		if tokenStr == "" {
			http.Error(w, "Missing token", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid claims", http.StatusUnauthorized)
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			http.Error(w, "Missing user_id in token", http.StatusUnauthorized)
			return
		}
		userID := int64(userIDFloat)

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ハンドラー内で userID を取得する関数
func GetUserID(r *http.Request) int64 {
	if val, ok := r.Context().Value(UserIDKey).(int64); ok {
		return val
	}
	return 0 // 存在しない・不正な場合は 0（認証失敗などで弾くのが良い）
}
