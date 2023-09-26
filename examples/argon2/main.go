package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log/slog"
	"time"

	"golang.org/x/crypto/argon2"
)

func main() {
	password := "123aBc"
	run(password)
}

func run(password string) {
	functions := make([]func(), 20)
	for i := 0; i < len(functions); i++ {
		t0 := time.Now()
		salt := salt()
		hash := argon(password, salt)
		saltStr := EncodeToString(salt)
		hashStr := EncodeToString(hash)
		slog.Info("generate",
			"[1]salt", saltStr,
			"[2]hash", hashStr,
			"[3]takes", time.Now().Sub(t0),
		)
		functions[i] = func() {
			verify(password, saltStr, hashStr)
		}
	}

	for _, fn := range functions {
		fn()
	}
}

func salt() []byte {
	arr := make([]byte, 32)
	_, _ = rand.Read(arr)
	sha := sha256.Sum224(arr)
	return sha[:12]
}

func argon(password string, salt []byte) []byte {
	keyLen := uint32(2 * len(salt))
	return argon2.IDKey([]byte(password), salt, 10, 2048, 1, keyLen)
}

func verify(password, saltStr, expectStr string) {
	salt := DecodeString(saltStr)
	hash := argon(password, salt)
	slog.Info("verify",
		"[1]salt", EncodeToString(salt),
		"[2]hash", EncodeToString(hash),
		"[3]expect", expectStr,
	)
}

func EncodeToString(ba []byte) string {
	return base64.URLEncoding.EncodeToString(ba)
}

func DecodeString(s string) []byte {
	ba, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		slog.Error(err.Error())
	}
	return ba
}
