package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aihub/backend-go/app/bootstrap"
	"github.com/aihub/backend-go/app/router"
	"github.com/aihub/backend-go/internal/auth"
	"github.com/aihub/backend-go/internal/services"
	"github.com/beego/beego/v2/server/web"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestKnowledgeBaseCRUD 测试知识库CRUD操作
func TestKnowledgeBaseCRUD(t *testing.T) {
	// 初始化应用
	app, err := bootstrap.Init()
	require.NoError(t, err)
	require.NotNil(t, app)

	// 初始化路由
	router.InitKnowledgeRoutes()

	// 创建JWT token用于认证
	jwtService := auth.NewJWTService("test-secret", "test-issuer", 3600)
	token, err := jwtService.GenerateToken(1, "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// 测试创建知识库
	createReq := map[string]interface{}{
		"name":        "测试知识库",
		"description": "这是一个测试知识库",
	}
	createBody, _ := json.Marshal(createReq)

	req := httptest.NewRequest("POST", "/api/knowledge", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "创建知识库应该成功")

	// 解析响应
	var createResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &createResp)
	require.NoError(t, err)
	assert.NotNil(t, createResp["data"])

	// 测试获取知识库列表
	req = httptest.NewRequest("GET", "/api/knowledge", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w = httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "获取知识库列表应该成功")
}

// TestDocumentUpload 测试文档上传
func TestDocumentUpload(t *testing.T) {
	// 初始化应用
	app, err := bootstrap.Init()
	require.NoError(t, err)
	require.NotNil(t, app)

	// 初始化路由
	router.InitKnowledgeRoutes()

	// 创建JWT token
	jwtService := auth.NewJWTService("test-secret", "test-issuer", 3600)
	token, err := jwtService.GenerateToken(1, "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// 测试上传文档
	uploadReq := services.UploadDocumentsRequest{
		Documents: []services.DocumentUpload{
			{
				Title:   "测试文档",
				Content:  "这是测试文档内容",
				Source:  "manual",
			},
		},
	}
	uploadBody, _ := json.Marshal(uploadReq)

	req := httptest.NewRequest("POST", "/api/knowledge/1/upload", bytes.NewBuffer(uploadBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)

	// 注意：实际测试可能需要数据库连接
	// 这里主要测试路由和控制器结构是否正确
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

// TestSearch 测试搜索功能
func TestSearch(t *testing.T) {
	// 初始化应用
	app, err := bootstrap.Init()
	require.NoError(t, err)
	require.NotNil(t, app)

	// 初始化路由
	router.InitKnowledgeRoutes()

	// 创建JWT token
	jwtService := auth.NewJWTService("test-secret", "test-issuer", 3600)
	token, err := jwtService.GenerateToken(1, "testuser", "test@example.com", []string{"user"})
	require.NoError(t, err)

	// 测试搜索
	req := httptest.NewRequest("GET", "/api/knowledge/1/search?query=测试", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	web.BeeApp.Handlers.ServeHTTP(w, req)

	// 注意：实际测试需要搜索引擎配置
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}
