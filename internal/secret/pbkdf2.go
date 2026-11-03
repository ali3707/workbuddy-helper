package secret

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
)

// PBKDF2SHA256 是 RFC 8018 PBKDF2-HMAC-SHA256 的最小实现，
// 用于口令派生（面板密码与账号备份），避免引入额外依赖。
func PBKDF2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	hashLen := sha256.Size
	blocks := (keyLen + hashLen - 1) / hashLen
	out := make([]byte, 0, blocks*hashLen)
	var index [4]byte
	for block := 1; block <= blocks; block++ {
		binary.BigEndian.PutUint32(index[:], uint32(block))
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write(index[:])
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for i := 1; i < iterations; i++ {
			mac.Reset()
			mac.Write(u)
			u = mac.Sum(u[:0])
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}
