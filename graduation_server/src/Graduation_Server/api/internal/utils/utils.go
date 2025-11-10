package utils

import (
	"math/rand"
	"time"
)

const roomCodeCharset = "0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateRoomCode() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = roomCodeCharset[rand.Intn(len(roomCodeCharset))]
	}
	return string(b)
}
