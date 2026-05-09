// Package monitor provides helper utilities shared by monitor types.
package monitor

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

const sensitiveKeyPatterns = "password,token,secret,api_key,authorization,dsn"

// GeneratePushToken creates a high-entropy random push token.
func GeneratePushToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken returns the SHA-256 hex hash of a token.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// MaskConfig takes a monitor config JSON and returns a copy with sensitive fields masked.
// Sensitive fields are those whose key name contains any of the patterns in sensitiveKeyPatterns.
func MaskConfig(config []byte) []byte {
	if len(config) == 0 {
		return config
	}
	var m map[string]any
	if err := json.Unmarshal(config, &m); err != nil {
		return config
	}
	masked := maskMap(m)
	out, _ := json.Marshal(masked)
	return out
}

func maskMap(m map[string]any) map[string]any {
	patterns := strings.Split(sensitiveKeyPatterns, ",")
	out := make(map[string]any, len(m))
	for k, v := range m {
		kLower := strings.ToLower(k)
		isSensitive := false
		for _, p := range patterns {
			if strings.Contains(kLower, p) {
				isSensitive = true
				break
			}
		}
		if isSensitive {
			out[k] = "***"
		} else if nested, ok := v.(map[string]any); ok {
			out[k] = maskMap(nested)
		} else {
			out[k] = v
		}
	}
	return out
}
