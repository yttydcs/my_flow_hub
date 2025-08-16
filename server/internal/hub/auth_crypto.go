package hub

import (
	"crypto/hmac"
	"crypto/sha256"
)

// computeHMACSHA256 计算 HMAC-SHA256(key, chunks...)
func computeHMACSHA256(key []byte, chunks ...[]byte) [32]byte {
	h := hmac.New(sha256.New, key)
	for _, c := range chunks {
		if len(c) > 0 {
			_, _ = h.Write(c)
		}
	}
	var out [32]byte
	sum := h.Sum(nil)
	copy(out[:], sum)
	return out
}

func hmacEqual(a []byte, b [32]byte) bool {
	if len(a) != 32 {
		return false
	}
	return hmac.Equal(a, b[:])
}
