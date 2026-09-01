// Package auth 注册 / 登录 / JWT 签发。
package auth

import (
	"errors"
	"regexp"
	"strings"

	"cryptosim/internal/ledger"
	"cryptosim/internal/model"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrBadEmail       = errors.New("邮箱格式不正确")
	ErrBadUsername    = errors.New("用户名需为 3-20 位字母、数字或下划线")
	ErrBadPassword    = errors.New("密码至少 8 位")
	ErrEmailTaken     = errors.New("该邮箱已被注册")
	ErrUsernameTaken  = errors.New("该用户名已被注册")
	ErrBadCredentials = errors.New("账号或密码错误")
	ErrDisabled       = errors.New("账号已被禁用")
)

var (
	emailRe    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
)

// 用户角色。
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type RegisterInput struct {
	Email    string
	Username string
	Password string
}

func ValidateRegister(in RegisterInput) error {
	if !emailRe.MatchString(in.Email) {
		return ErrBadEmail
	}
	if !usernameRe.MatchString(in.Username) {
		return ErrBadUsername
	}
	if len(in.Password) < 8 {
		return ErrBadPassword
	}
	return nil
}

type Service struct {
	db          *gorm.DB
	initialUSDT decimal.Decimal
	cryptor     SecretCryptor
}

func NewService(db *gorm.DB, initialUSDT string, cryptor SecretCryptor) *Service {
	return &Service{db: db, initialUSDT: decimal.RequireFromString(initialUSDT), cryptor: cryptor}
}

// IssueCredentials 为用户签发（或轮换）HMAC 凭证对，secret 明文仅此一次返回。
func (s *Service) IssueCredentials(uid int64) (apiKey, apiSecret string, err error) {
	return issueCredentials(s.db, s.cryptor, uid)
}

// ResetCredentials 重置 HMAC 凭证（旧凭证立即失效）。
func (s *Service) ResetCredentials(uid int64) (apiKey, apiSecret string, err error) {
	return issueCredentials(s.db, s.cryptor, uid)
}

// writeLoginLog 记录登录审计（成功与失败都记）。
func (s *Service) writeLoginLog(userID *int64, username, reason string, success bool, ip, ua string) {
	_ = s.db.Create(&model.LoginLog{
		UserID: userID, Username: username, Success: success,
		Reason: reason, IP: ip, UserAgent: ua,
	}).Error
}

// Register 创建用户并在同一事务内发放初始虚拟 USDT，随后签发 HMAC 凭证。
func (s *Service) Register(in RegisterInput, ip, ua string) (*model.User, string, string, error) {
	if err := ValidateRegister(in); err != nil {
		return nil, "", "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", "", err
	}
	u := &model.User{
		Email:        strings.ToLower(strings.TrimSpace(in.Email)),
		Username:     in.Username,
		PasswordHash: string(hash),
		Role:         RoleUser, // 显式赋值，避免 GORM 写空串覆盖数据库默认值
		Status:       1,
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return mapDup(err)
		}
		return ledger.GrantUSDT(tx, u.ID, s.initialUSDT, "signup_grant", "", "注册赠送虚拟资金")
	})
	if err != nil {
		return nil, "", "", err
	}
	apiKey, apiSecret, err := s.IssueCredentials(u.ID)
	if err != nil {
		return nil, "", "", err
	}
	s.writeLoginLog(&u.ID, u.Username, "注册", true, ip, ua)
	return u, apiKey, apiSecret, nil
}

// SeedDemoUsers 幂等创建文档承诺的内置演示账号 demo / admin（各赠初始虚拟 USDT 并签发 HMAC 凭证）。
// 仅在非生产环境调用：本地联调与 e2e 测试依赖这两个账号，生产环境不应存在公开口令账号。
func (s *Service) SeedDemoUsers() error {
	demos := []struct {
		Email, Username, Password, Role string
	}{
		{"demo@cryptosim.local", "demo", "demo12345", RoleUser},
		{"admin@cryptosim.local", "admin", "admin12345", RoleAdmin},
	}
	for _, d := range demos {
		var n int64
		if err := s.db.Model(&model.User{}).Where("username = ?", d.Username).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 { // 已存在则跳过，不改密码不重复发币
			continue
		}
		u, _, _, err := s.Register(
			RegisterInput{Email: d.Email, Username: d.Username, Password: d.Password},
			"127.0.0.1", "seed",
		)
		if err != nil {
			return err
		}
		if err := s.db.Model(&model.User{}).Where("id = ?", u.ID).Update("role", d.Role).Error; err != nil {
			return err
		}
	}
	return nil
}

// Login 支持邮箱或用户名登录，成功后轮换 HMAC 凭证。
func (s *Service) Login(account, password, ip, ua string) (*model.User, string, string, error) {
	account = strings.TrimSpace(account)
	if account == "" || password == "" {
		s.writeLoginLog(nil, account, "参数为空", false, ip, ua)
		return nil, "", "", ErrBadCredentials
	}
	var u model.User
	err := s.db.Where("email = ? OR username = ?", strings.ToLower(account), account).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s.writeLoginLog(nil, account, "账号不存在", false, ip, ua)
		return nil, "", "", ErrBadCredentials
	}
	if err != nil {
		return nil, "", "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		s.writeLoginLog(&u.ID, u.Username, "密码错误", false, ip, ua)
		return nil, "", "", ErrBadCredentials
	}
	if u.Status != 1 {
		s.writeLoginLog(&u.ID, u.Username, "账号被禁用", false, ip, ua)
		return nil, "", "", ErrDisabled
	}
	apiKey, apiSecret, err := s.IssueCredentials(u.ID)
	if err != nil {
		return nil, "", "", err
	}
	s.writeLoginLog(&u.ID, u.Username, "登录成功", true, ip, ua)
	return &u, apiKey, apiSecret, nil
}

// mapDup 将 PG 唯一约束冲突翻译为业务错误。
func mapDup(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_email_key":
			return ErrEmailTaken
		case "users_username_key":
			return ErrUsernameTaken
		}
	}
	return err
}
