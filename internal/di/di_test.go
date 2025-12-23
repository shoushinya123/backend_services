package di

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyInjectionContainer(t *testing.T) {
	// 初始化DI容器
	container := InitContainer()
	assert.NotNil(t, container)

	// 验证容器已创建
	assert.NotNil(t, Container)

	t.Log("DI container initialization test passed!")
}

func TestContainerBasicOperations(t *testing.T) {
	container := InitContainer()

	// 测试基本的Provide操作
	type TestService struct {
		Name string
	}

	err := container.Provide(func() *TestService {
		return &TestService{Name: "test"}
	})
	require.NoError(t, err)

	// 测试基本的Invoke操作
	err = container.Invoke(func(svc *TestService) {
		assert.Equal(t, "test", svc.Name)
	})
	assert.NoError(t, err)

	t.Log("DI container basic operations test passed!")
}
