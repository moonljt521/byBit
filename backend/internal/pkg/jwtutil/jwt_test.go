package jwtutil

import (
	"testing"
	"time"
)

const secret = "test-secret"

func TestGenerateParseRoundTrip(t *testing.T) {
	token, err := Generate(42, secret, time.Hour)
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	uid, err := Parse(token, secret)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}
	if uid != 42 {
		t.Fatalf("uid = %d, want 42", uid)
	}
}

func TestParseWrongSecret(t *testing.T) {
	token, _ := Generate(1, secret, time.Hour)
	if _, err := Parse(token, "other"); err == nil {
		t.Fatal("错误密钥应解析失败")
	}
}

func TestParseExpired(t *testing.T) {
	token, _ := Generate(1, secret, -time.Minute)
	if _, err := Parse(token, secret); err == nil {
		t.Fatal("过期 token 应解析失败")
	}
}
