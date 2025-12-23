package services

import (
	"testing"

	"github.com/aihub/backend-go/internal/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// MockDatabaseInterface 模拟数据库接口
type MockDatabaseInterface struct {
	mock.Mock
}

func (m *MockDatabaseInterface) GormDB() *gorm.DB {
	args := m.Called()
	return args.Get(0).(*gorm.DB)
}

func (m *MockDatabaseInterface) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDatabaseInterface) HealthCheck() error {
	args := m.Called()
	return args.Error(0)
}

// MockLoggerInterface 模拟日志接口
type MockLoggerInterface struct {
	mock.Mock
}

func (m *MockLoggerInterface) Debug(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLoggerInterface) Info(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLoggerInterface) Warn(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLoggerInterface) Error(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLoggerInterface) Fatal(msg string, fields ...interface{}) {
	m.Called(msg, fields)
}

func (m *MockLoggerInterface) With(fields ...interface{}) interfaces.LoggerInterface {
	args := m.Called(fields)
	return args.Get(0).(interfaces.LoggerInterface)
}

func (m *MockLoggerInterface) WithError(err error) interfaces.LoggerInterface {
	args := m.Called(err)
	return args.Get(0).(interfaces.LoggerInterface)
}

func TestNewKnowledgeBaseService(t *testing.T) {
	mockDB := new(MockDatabaseInterface)
	mockLogger := new(MockLoggerInterface)

	service := NewKnowledgeBaseService(mockDB, mockLogger)
	assert.NotNil(t, service)
	assert.Equal(t, mockDB, service.db)
	assert.Equal(t, mockLogger, service.logger)
}

// 注意：完整的测试需要真实的数据库连接或更完善的mock
// 这里只是展示测试结构
