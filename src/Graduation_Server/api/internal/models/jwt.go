package models

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTの署名に使用する秘密鍵（環境変数などで安全に管理することが望ましい）
var jwtSecret = []byte("your-secret-key")

// JWTを生成する関数
// 引数: userID - ユーザー識別子
// 戻り値: 署名されたJWT文字列とエラー
func GenerateJWT(userID string) (string, error) {
	// クレーム（Claims）を設定：userIDと有効期限（24時間）
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,                                // ユーザーIDをクレームに含める
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // 有効期限を現在から24時間後に設定
	})

	// 秘密鍵を使ってトークンに署名し、文字列として返す
	return token.SignedString(jwtSecret)
}

// JWTを検証し、有効であればuserIDを返す関数
// 引数: tokenStr - クライアントから送られたJWT文字列
// 戻り値: userIDとエラー
func ValidateJWT(tokenStr string) (string, error) {
	// トークンのパースと署名検証
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		// 使用する署名アルゴリズムと秘密鍵を返す
		return jwtSecret, nil
	})

	// トークンが有効かつ、クレームの型がMapClaims（汎用形式）として取得できた場合
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// クレームからuser_idを取得して返す
		if userID, ok := claims["user_id"].(string); ok {
			return userID, nil
		}
	}

	// 検証失敗時は空文字とエラーを返す
	return "", err
}
