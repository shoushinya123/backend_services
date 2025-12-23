package services

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreakerState 熔断器状态
type CircuitBreakerState int32

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	name string

	// 配置
	failureThreshold int           // 失败阈值
	successThreshold int           // 成功阈值（半开状态）
	timeout          time.Duration // 熔断超时时间

	// 状态
	state           int32
	failureCount    int32
	successCount    int32
	lastFailureTime time.Time
	mutex           sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(name string, failureThreshold int, successThreshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            int32(StateClosed),
	}
}

// Call 执行函数调用（带熔断保护）
func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.canExecute() {
		return &CircuitBreakerError{
			Name:  cb.name,
			State: cb.getState(),
			Err:   ErrCircuitOpen,
		}
	}

	err := fn()
	cb.recordResult(err == nil)

	if err != nil {
		return &CircuitBreakerError{
			Name:  cb.name,
			State: cb.getState(),
			Err:   err,
		}
	}

	return nil
}

// canExecute 检查是否可以执行请求
func (cb *CircuitBreaker) canExecute() bool {
	state := cb.getState()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// 检查是否可以尝试半开
		cb.mutex.RLock()
		canHalfOpen := time.Since(cb.lastFailureTime) >= cb.timeout
		cb.mutex.RUnlock()

		if canHalfOpen {
			atomic.StoreInt32(&cb.state, int32(StateHalfOpen))
			atomic.StoreInt32(&cb.successCount, 0)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult 记录执行结果
func (cb *CircuitBreaker) recordResult(success bool) {
	if success {
		cb.recordSuccess()
	} else {
		cb.recordFailure()
	}
}

// recordSuccess 记录成功
func (cb *CircuitBreaker) recordSuccess() {
	state := cb.getState()

	switch state {
	case StateHalfOpen:
		// 半开状态下，成功计数增加
		count := atomic.AddInt32(&cb.successCount, 1)
		if int(count) >= cb.successThreshold {
			// 达到成功阈值，关闭熔断器
			atomic.StoreInt32(&cb.state, int32(StateClosed))
			atomic.StoreInt32(&cb.failureCount, 0)
		}
	case StateClosed:
		// 关闭状态下，重置失败计数
		atomic.StoreInt32(&cb.failureCount, 0)
	}
}

// recordFailure 记录失败
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	cb.lastFailureTime = time.Now()
	cb.mutex.Unlock()

	state := cb.getState()

	switch state {
	case StateHalfOpen:
		// 半开状态下失败，直接打开熔断器
		atomic.StoreInt32(&cb.state, int32(StateOpen))
		atomic.StoreInt32(&cb.successCount, 0)
	case StateClosed:
		// 关闭状态下，失败计数增加
		count := atomic.AddInt32(&cb.failureCount, 1)
		if int(count) >= cb.failureThreshold {
			// 达到失败阈值，打开熔断器
			atomic.StoreInt32(&cb.state, int32(StateOpen))
		}
	}
}

// getState 获取当前状态
func (cb *CircuitBreaker) getState() CircuitBreakerState {
	return CircuitBreakerState(atomic.LoadInt32(&cb.state))
}

// GetState 获取当前状态（外部接口）
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.getState()
}

// GetStats 获取统计信息
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"name":              cb.name,
		"state":             cb.getState().String(),
		"failure_count":     atomic.LoadInt32(&cb.failureCount),
		"success_count":     atomic.LoadInt32(&cb.successCount),
		"failure_threshold": cb.failureThreshold,
		"success_threshold": cb.successThreshold,
		"timeout":           cb.timeout.String(),
		"last_failure_time": cb.lastFailureTime,
	}
}

// String 返回状态字符串
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerError 熔断器错误
type CircuitBreakerError struct {
	Name  string
	State CircuitBreakerState
	Err   error
}

func (e *CircuitBreakerError) Error() string {
	return e.Err.Error()
}

func (e *CircuitBreakerError) Unwrap() error {
	return e.Err
}

// 预定义错误
var (
	ErrCircuitOpen = &CircuitBreakerError{Err: &CircuitOpenError{}}
)

// CircuitOpenError 熔断器打开错误
type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open"
}

// GlobalCircuitBreakers 全局熔断器管理
var (
	globalCircuitBreakers = make(map[string]*CircuitBreaker)
	circuitBreakerMutex   sync.RWMutex
)

// GetCircuitBreaker 获取或创建全局熔断器
func GetCircuitBreaker(name string) *CircuitBreaker {
	circuitBreakerMutex.RLock()
	cb, exists := globalCircuitBreakers[name]
	circuitBreakerMutex.RUnlock()

	if exists {
		return cb
	}

	// 创建新的熔断器
	cb = NewCircuitBreaker(name, 5, 3, time.Minute*1)

	circuitBreakerMutex.Lock()
	globalCircuitBreakers[name] = cb
	circuitBreakerMutex.Unlock()

	return cb
}

// GetAllCircuitBreakers 获取所有熔断器状态
func GetAllCircuitBreakers() map[string]interface{} {
	circuitBreakerMutex.RLock()
	defer circuitBreakerMutex.RUnlock()

	result := make(map[string]interface{})
	for name, cb := range globalCircuitBreakers {
		result[name] = cb.GetStats()
	}

	return result
}

