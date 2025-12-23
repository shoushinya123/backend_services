package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "test-issuer", time.Hour)

	token, err := service.GenerateToken(1, "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTService_ValidateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", "test-issuer", time.Hour)

	// 生成token
	token, err := service.GenerateToken(1, "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// 验证token
	claims, err := service.ValidateToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint(1), claims.UserID)
	assert.Equal(t, "testuser", claims.Username)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, []string{"user"}, claims.Roles)
}

func TestJWTService_ValidateToken_Expired(t *testing.T) {
	service := NewJWTService("test-secret-key", "test-issuer", -time.Hour) // 已过期

	token, err := service.GenerateToken(1, "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// 验证过期token
	_, err = service.ValidateToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestJWTService_ValidateToken_Invalid(t *testing.T) {
	service := NewJWTService("test-secret-key", "test-issuer", time.Hour)

	// 使用错误的密钥验证
	wrongService := NewJWTService("wrong-secret-key", "test-issuer", time.Hour)
	token, err := wrongService.GenerateToken(1, "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// 使用正确的服务验证错误的token
	_, err = service.ValidateToken(token)
	assert.Error(t, err)
}

func TestExtractTokenFromHeader(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		want    string
		wantErr bool
	}{
		{
			name:    "valid token",
			header:  "Bearer valid-token",
			want:    "valid-token",
			wantErr: false,
		},
		{
			name:    "empty header",
			header:  "",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing bearer prefix",
			header:  "valid-token",
			want:    "",
			wantErr: true,
		},
		{
			name:    "empty token",
			header:  "Bearer ",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := ExtractTokenFromHeader(tt.header)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, token)
			}
		})
	}
}
