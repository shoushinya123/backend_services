package v2

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// EncryptionService 配置加密服务
type EncryptionService struct {
	key []byte
}

// NewEncryptionService 创建加密服务
func NewEncryptionService(masterKey string) (*EncryptionService, error) {
	if masterKey == "" {
		// 尝试从环境变量获取
		masterKey = os.Getenv("CONFIG_ENCRYPTION_KEY")
		if masterKey == "" {
			// 生成随机密钥（仅用于开发环境）
			key := make([]byte, 32)
			if _, err := rand.Read(key); err != nil {
				return nil, fmt.Errorf("failed to generate encryption key: %w", err)
			}
			fmt.Println("Warning: Using randomly generated encryption key. Set CONFIG_ENCRYPTION_KEY for production.")
			return &EncryptionService{key: key}, nil
		}
	}

	// 从密码短语生成密钥
	key, err := deriveKey(masterKey, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	return &EncryptionService{key: key}, nil
}

// deriveKey 从密码短语派生密钥
func deriveKey(password string, keyLen int) ([]byte, error) {
	// 使用bcrypt派生密钥
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 截取到所需长度
	if len(hash) < keyLen {
		return nil, fmt.Errorf("derived key too short")
	}

	return hash[:keyLen], nil
}

// Encrypt 加密数据
func (es *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(es.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密数据
func (es *EncryptionService) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(es.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// IsEncrypted 检查字符串是否已加密
func (es *EncryptionService) IsEncrypted(s string) bool {
	if s == "" {
		return false
	}

	// 检查是否是base64编码的格式（简单检查）
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil && strings.Contains(s, "=") // base64通常包含等号
}

// EncryptConfig 加密配置中的敏感字段
func (es *EncryptionService) EncryptConfig(config *ConfigV2) error {
	// 加密数据库密码（如果URL看起来像是包含密码的）
	if strings.Contains(config.Database.URL, "@") &&
	   (strings.Contains(config.Database.URL, "password") ||
	    strings.Contains(config.Database.URL, "pass")) {
		encrypted, err := es.Encrypt(config.Database.URL)
		if err != nil {
			return fmt.Errorf("failed to encrypt database URL: %w", err)
		}
		config.Database.URL = "encrypted:" + encrypted
	}

	// 加密缓存密码
	if config.Cache.Password != "" {
		encrypted, err := es.Encrypt(config.Cache.Password)
		if err != nil {
			return fmt.Errorf("failed to encrypt cache password: %w", err)
		}
		config.Cache.Password = "encrypted:" + encrypted
	}

	// 加密认证密钥
	if config.Auth.Secret != "" && !strings.HasPrefix(config.Auth.Secret, "encrypted:") {
		encrypted, err := es.Encrypt(config.Auth.Secret)
		if err != nil {
			return fmt.Errorf("failed to encrypt auth secret: %w", err)
		}
		config.Auth.Secret = "encrypted:" + encrypted
	}

	// 加密AI API密钥
	if config.AI.DashScopeAPIKey != "" && !strings.HasPrefix(config.AI.DashScopeAPIKey, "encrypted:") {
		encrypted, err := es.Encrypt(config.AI.DashScopeAPIKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt AI API key: %w", err)
		}
		config.AI.DashScopeAPIKey = "encrypted:" + encrypted
	}

	// 加密存储密钥
	if config.Storage.SecretKey != "" && !strings.HasPrefix(config.Storage.SecretKey, "encrypted:") {
		encrypted, err := es.Encrypt(config.Storage.SecretKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt storage secret key: %w", err)
		}
		config.Storage.SecretKey = "encrypted:" + encrypted
	}

	return nil
}

// DecryptConfig 解密配置中的敏感字段
func (es *EncryptionService) DecryptConfig(config *ConfigV2) error {
	// 解密数据库URL
	if strings.HasPrefix(config.Database.URL, "encrypted:") {
		encrypted := strings.TrimPrefix(config.Database.URL, "encrypted:")
		decrypted, err := es.Decrypt(encrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt database URL: %w", err)
		}
		config.Database.URL = decrypted
	}

	// 解密缓存密码
	if strings.HasPrefix(config.Cache.Password, "encrypted:") {
		encrypted := strings.TrimPrefix(config.Cache.Password, "encrypted:")
		decrypted, err := es.Decrypt(encrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt cache password: %w", err)
		}
		config.Cache.Password = decrypted
	}

	// 解密认证密钥
	if strings.HasPrefix(config.Auth.Secret, "encrypted:") {
		encrypted := strings.TrimPrefix(config.Auth.Secret, "encrypted:")
		decrypted, err := es.Decrypt(encrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt auth secret: %w", err)
		}
		config.Auth.Secret = decrypted
	}

	// 解密AI API密钥
	if strings.HasPrefix(config.AI.DashScopeAPIKey, "encrypted:") {
		encrypted := strings.TrimPrefix(config.AI.DashScopeAPIKey, "encrypted:")
		decrypted, err := es.Decrypt(encrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt AI API key: %w", err)
		}
		config.AI.DashScopeAPIKey = decrypted
	}

	// 解密存储密钥
	if strings.HasPrefix(config.Storage.SecretKey, "encrypted:") {
		encrypted := strings.TrimPrefix(config.Storage.SecretKey, "encrypted:")
		decrypted, err := es.Decrypt(encrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt storage secret key: %w", err)
		}
		config.Storage.SecretKey = decrypted
	}

	return nil
}
