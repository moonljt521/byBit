package model

import "time"

// ApiCredential HMAC 验签凭证对：apiSecret 仅签发时明文返回一次，落库为 AES-GCM 密文。
type ApiCredential struct {
	UserID          int64     `gorm:"primaryKey" json:"user_id"`
	ApiKey          string    `json:"api_key"`
	SecretEncrypted string    `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (ApiCredential) TableName() string { return "api_credentials" }

// LoginLog 登录审计日志。
type LoginLog struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    *int64    `json:"user_id"`
	Username  string    `json:"username"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

func (LoginLog) TableName() string { return "login_logs" }
