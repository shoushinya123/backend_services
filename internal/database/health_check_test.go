package database

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthChecker_Basic(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 设置mock期望ping成功
	mock.ExpectPing()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	checker := NewHealthChecker(db, logger)
	assert.NotNil(t, checker)

	// 测试健康检查
	ctx := context.Background()
	err = checker.Check(ctx)
	assert.NoError(t, err)
	assert.True(t, checker.IsHealthy())

	// 验证mock期望
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthChecker_FailureAndRecovery(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	checker := NewHealthChecker(db, logger)

	// 设置ping失败
	mock.ExpectPing().WillReturnError(sqlmock.ErrCancelled)

	// 测试失败检查
	ctx := context.Background()
	err = checker.Check(ctx)
	assert.Error(t, err)
	assert.False(t, checker.IsHealthy())

	// 设置ping成功
	mock.ExpectPing()

	// 测试恢复
	err = checker.Check(ctx)
	assert.NoError(t, err)
	assert.True(t, checker.IsHealthy())

	// 验证mock期望
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthChecker_BackgroundMonitoring(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// 设置ping成功
	mock.ExpectPing()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	checker := NewHealthChecker(db, logger)
	checker.SetCheckInterval(100 * time.Millisecond) // 快速检查间隔

	// 启动健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	checker.Start(ctx)

	// 等待一段时间让检查执行
	time.Sleep(300 * time.Millisecond)

	// 停止检查
	checker.Stop()

	// 验证状态
	assert.True(t, checker.IsHealthy())

	// 验证mock期望
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthChecker_Result(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	checker := NewHealthChecker(db, logger)

	// 测试初始状态
	result := checker.GetHealthResult()
	assert.False(t, result.Healthy)
	assert.NotZero(t, result.LastCheck)

	// 设置ping成功
	mock.ExpectPing()

	// 执行检查
	ctx := context.Background()
	err = checker.Check(ctx)
	require.NoError(t, err)

	// 获取结果
	result = checker.GetHealthResult()
	assert.True(t, result.Healthy)
	assert.Empty(t, result.LastError)

	// 验证mock期望
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHealthChecker_WaitForHealthy(t *testing.T) {
	// 创建mock数据库
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	checker := NewHealthChecker(db, logger)

	// 先设置失败状态
	mock.ExpectPing().WillReturnError(sqlmock.ErrCancelled)
	ctx := context.Background()
	err = checker.Check(ctx)
	require.Error(t, err)

	// 设置恢复
	mock.ExpectPing()

	// 启动goroutine模拟恢复
	go func() {
		time.Sleep(50 * time.Millisecond)
		checker.Check(ctx)
	}()

	// 等待健康
	timeoutCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	err = checker.WaitForHealthy(timeoutCtx, 150*time.Millisecond)
	assert.NoError(t, err)
	assert.True(t, checker.IsHealthy())

	// 验证mock期望
	assert.NoError(t, mock.ExpectationsWereMet())
}
