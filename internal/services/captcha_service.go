package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/redis/go-redis/v9"
)

// CaptchaService 验证码服务
type CaptchaService struct {
	redis *redis.Client
}

// NewCaptchaService 创建验证码服务
func NewCaptchaService() *CaptchaService {
	return &CaptchaService{
		redis: database.RedisClient,
	}
}

// GenerateCaptcha 生成验证码（返回验证码字符串和图片数据）
func (s *CaptchaService) GenerateCaptcha(key string) (string, []byte, error) {
	// 生成4位随机验证码（数字+字母）
	code := generateRandomCode(4)
	
	// 存储到Redis（5分钟有效）
	if s.redis != nil {
		ctx := context.Background()
		redisKey := "auth:v2:captcha:" + key
		s.redis.Set(ctx, redisKey, strings.ToUpper(code), 5*time.Minute)
	}

	// 生成图片
	img := generateCaptchaImage(code)
	
	// 编码为PNG
	imgData, err := encodeImageToPNG(img)
	if err != nil {
		return "", nil, err
	}

	// 转换为base64编码（前端需要）
	imgBase64 := base64.StdEncoding.EncodeToString(imgData)

	return code, []byte(imgBase64), nil
}

// ValidateCaptcha 验证验证码
func (s *CaptchaService) ValidateCaptcha(key, code string) bool {
	if s.redis == nil {
		return true // Redis未初始化时跳过验证
	}

	ctx := context.Background()
	redisKey := "auth:v2:captcha:" + key
	storedCode, err := s.redis.Get(ctx, redisKey).Result()
	if err != nil {
		return false
	}

	// 验证码不区分大小写
	if strings.ToUpper(storedCode) != strings.ToUpper(code) {
		return false
	}

	// 验证成功后删除
	s.redis.Del(ctx, redisKey)
	return true
}

// generateRandomCode 生成随机验证码
func generateRandomCode(length int) string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 排除易混淆字符
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}

// generateCaptchaImage 生成验证码图片
func generateCaptchaImage(code string) *image.RGBA {
	width := 120
	height := 40
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// 填充背景色（浅灰色）
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{240, 240, 240, 255})
		}
	}

	// 绘制验证码文字（简化版，实际应该使用字体）
	// 这里只是示例，实际应该使用字体库绘制
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i, char := range code {
		x := 20 + i*25
		y := 25 + r.Intn(10)
		// 随机颜色
		c := color.RGBA{
			uint8(50 + r.Intn(150)),
			uint8(50 + r.Intn(150)),
			uint8(50 + r.Intn(150)),
			255,
		}
		// 简化：绘制一个矩形代表字符（实际应该绘制文字）
		drawChar(img, x, y, string(char), c)
	}

	// 添加干扰线
	for i := 0; i < 3; i++ {
		x1 := r.Intn(width)
		y1 := r.Intn(height)
		x2 := r.Intn(width)
		y2 := r.Intn(height)
		drawLine(img, x1, y1, x2, y2, color.RGBA{200, 200, 200, 255})
	}

	return img
}

// drawChar 绘制字符（简化版）
func drawChar(img *image.RGBA, x, y int, char string, c color.RGBA) {
	// 简化处理：绘制一个矩形
	for i := 0; i < 15; i++ {
		for j := 0; j < 15; j++ {
			if x+i < img.Bounds().Dx() && y+j < img.Bounds().Dy() {
				img.Set(x+i, y+j, c)
			}
		}
	}
}

// drawLine 绘制线条
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	if x1 > x2 {
		sx = -1
	}
	sy := 1
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	x, y := x1, y1
	for {
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
		if x == x2 && y == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x += sx
		}
		if e2 < dx {
			err += dx
			y += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// encodeImageToPNG 将图片编码为PNG
func encodeImageToPNG(img *image.RGBA) ([]byte, error) {
	// 使用bytes.Buffer和png.Encode
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

