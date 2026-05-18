package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const defaultPassword = "lingdang666"

var (
	tokenStore   = make(map[string]time.Time)
	tokenStoreMu sync.RWMutex
	tokenTTL     = 24 * time.Hour
)

// GenerateToken 生成随机 token
func GenerateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// SaveToken 保存 token
func SaveToken(token string) {
	tokenStoreMu.Lock()
	defer tokenStoreMu.Unlock()
	tokenStore[token] = time.Now().Add(tokenTTL)
}

// ValidateToken 验证 token 是否有效
func ValidateToken(token string) bool {
	tokenStoreMu.RLock()
	defer tokenStoreMu.RUnlock()
	expiry, ok := tokenStore[token]
	return ok && time.Now().Before(expiry)
}

// CleanupTokens 清理过期 token
func CleanupTokens() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		tokenStoreMu.Lock()
		now := time.Now()
		for t, exp := range tokenStore {
			if now.After(exp) {
				delete(tokenStore, t)
			}
		}
		tokenStoreMu.Unlock()
	}
}

// AuthMiddleware 认证中间件（对需要登录的接口使用）
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Header 获取 token
		token := r.Header.Get("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		// 也支持 query 参数
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if token == "" || !ValidateToken(token) {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"error": "未登录或 token 已过期",
			})
			return
		}

		next(w, r)
	}
}

// getPassword 获取密码（优先环境变量）
func getPassword() string {
	if p := getEnv("PASSWORD", ""); p != "" {
		return p
	}
	return defaultPassword
}
