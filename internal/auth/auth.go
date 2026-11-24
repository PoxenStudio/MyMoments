package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)
	errBadHash      = errors.New("invalid password hash")
)

// ValidateUsername 校验用户名格式。
func ValidateUsername(username string) bool {
	return usernamePattern.MatchString(username)
}

// ValidatePassword 校验口令长度。
func ValidatePassword(password string) error {
	if len([]rune(password)) < 10 {
		return errors.New("密码至少需要 10 个字符")
	}
	if len(password) > 256 {
		return errors.New("密码过长")
	}
	return nil
}

// HashPassword 使用 argon2id 生成可自我描述的口令哈希。
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	const (
		time    = 3
		memory  = 64 * 1024
		threads = 2
		keyLen  = 32
	)
	key := argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory, time, threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword 校验口令，使用常量时间比较。
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(expected, actual) == 1
}

// HashToken 返回令牌的 SHA-256 十六进制摘要，数据库只保存摘要。
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// RandomToken 生成 URL 安全的随机令牌。
func RandomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// GenerateTemporaryPassword 生成便于口述的临时密码。
func GenerateTemporaryPassword() (string, error) {
	const alphabet = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 14)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, len(buf))
	for i, b := range buf {
		out[i] = alphabet[int(b)%len(alphabet)]
	}
	return string(out), nil
}

// NormalizeUsername 去除空白并保持大小写签名，仅做裁剪。
func NormalizeUsername(v string) string { return strings.TrimSpace(v) }
