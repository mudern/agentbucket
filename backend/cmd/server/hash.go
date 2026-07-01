package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:7]
}

func hostPortFor(value string) int {
	sum := sha1.Sum([]byte(value))
	return 18000 + int(sum[0])%1000
}

func slug(value string) string {
	value = strings.ToLower(value)
	var buf bytes.Buffer
	lastDash := false
	for _, ch := range value {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if ok {
			buf.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			buf.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(buf.String(), "-")
}
