package auth

import "testing"

func TestValidateRegister(t *testing.T) {
	cases := []struct {
		name    string
		in      RegisterInput
		wantErr error
	}{
		{"合法", RegisterInput{"a@b.com", "alice_1", "password8"}, nil},
		{"邮箱格式错误", RegisterInput{"not-an-email", "alice", "password8"}, ErrBadEmail},
		{"用户名太短", RegisterInput{"a@b.com", "ab", "password8"}, ErrBadUsername},
		{"用户名含非法字符", RegisterInput{"a@b.com", "bad name!", "password8"}, ErrBadUsername},
		{"密码太短", RegisterInput{"a@b.com", "alice", "1234567"}, ErrBadPassword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRegister(tc.in)
			if err != tc.wantErr {
				t.Fatalf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}
