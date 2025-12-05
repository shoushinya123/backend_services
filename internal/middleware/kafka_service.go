package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/aihub/backend-go/internal/kafka"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// KafkaService Kafka事件服务
type KafkaService struct {
	producer *kafka.Producer
}

// NewKafkaService 创建Kafka服务实例
func NewKafkaService() *KafkaService {
	return &KafkaService{
		producer: kafka.GetProducer(),
	}
}

// EventMessage 标准事件消息格式
type EventMessage struct {
	EventType string                 `json:"event_type"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    *uint                  `json:"user_id,omitempty"`
	Data      interface{}            `json:"data"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TaskEvent 任务事件
type TaskEvent struct {
	TaskID   string                 `json:"task_id"`
	TaskType string                 `json:"task_type"` // crawler, processing
	Action   string                 `json:"action"`    // create, run, stop
	Config   map[string]interface{} `json:"config"`
	UserID   uint                   `json:"user_id"`
}

// WorkflowEvent 工作流事件
type WorkflowEvent struct {
	WorkflowID  string                 `json:"workflow_id"`
	ExecutionID string                 `json:"execution_id"`
	Action      string                 `json:"action"` // run, pause, stop
	Input       map[string]interface{} `json:"input"`
	UserID      uint                   `json:"user_id"`
}

// AIChatEvent AI聊天事件
type AIChatEvent struct {
	SessionID string                 `json:"session_id"`
	Message   string                 `json:"message"`
	ModelID   string                 `json:"model_id"`
	Context   map[string]interface{} `json:"context,omitempty"`
	UserID    uint                   `json:"user_id"`
}

// KnowledgeProcessEvent 知识库处理事件
type KnowledgeProcessEvent struct {
	KnowledgeBaseID uint   `json:"knowledge_base_id"`
	DocumentID      uint   `json:"document_id,omitempty"`
	Action          string `json:"action"` // process, index, sync
	UserID          uint   `json:"user_id"`
}

// SendMessage 发送消息到指定Topic
func (s *KafkaService) SendMessage(topic string, message interface{}) error {
	if s.producer == nil {
		logger.Warn("Kafka producer not initialized, skipping message")
		return nil
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	// 获取底层sarama producer
	saramaProducer := s.producer.GetProducerInstance()
	if saramaProducer == nil {
		logger.Warn("Sarama producer not available, skipping message")
		return nil
	}

	// 创建Kafka消息
	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(data),
	}

	// 发送消息
	partition, offset, err := saramaProducer.SendMessage(kafkaMsg)
	if err != nil {
		logger.Error("发送Kafka消息失败", zap.Error(err), zap.String("topic", topic))
		return fmt.Errorf("发送消息失败: %w", err)
	}

	logger.Debug("Kafka消息发送成功",
		zap.String("topic", topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset))

	return nil
}

// SendMessageWithKey 发送带key的消息
func (s *KafkaService) SendMessageWithKey(topic, key string, message interface{}) error {
	if s.producer == nil {
		return nil
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	// 获取底层sarama producer
	saramaProducer := s.producer.GetProducerInstance()
	if saramaProducer == nil {
		logger.Warn("Sarama producer not available, skipping message")
		return nil
	}

	// 创建Kafka消息（带key）
	kafkaMsg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(data),
	}

	// 发送消息
	partition, offset, err := saramaProducer.SendMessage(kafkaMsg)
	if err != nil {
		logger.Error("发送Kafka消息失败", zap.Error(err), zap.String("topic", topic), zap.String("key", key))
		return fmt.Errorf("发送消息失败: %w", err)
	}

	logger.Debug("Kafka消息发送成功",
		zap.String("topic", topic),
		zap.String("key", key),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset))

	return nil
}

// PublishTaskEvent 发布任务事件
func (s *KafkaService) PublishTaskEvent(event TaskEvent) error {
	topic := fmt.Sprintf("task.%s.%s", event.TaskType, event.Action)

	msg := EventMessage{
		EventType: fmt.Sprintf("task.%s.%s", event.TaskType, event.Action),
		Timestamp: time.Now(),
		UserID:    &event.UserID,
		Data:      event,
	}

	return s.SendMessage(topic, msg)
}

// PublishWorkflowEvent 发布工作流事件
func (s *KafkaService) PublishWorkflowEvent(event WorkflowEvent) error {
	topic := fmt.Sprintf("workflow.%s", event.Action)

	msg := EventMessage{
		EventType: fmt.Sprintf("workflow.%s", event.Action),
		Timestamp: time.Now(),
		UserID:    &event.UserID,
		Data:      event,
	}

	return s.SendMessage(topic, msg)
}

// PublishAIChatEvent 发布AI聊天事件
func (s *KafkaService) PublishAIChatEvent(event AIChatEvent) error {
	topic := "ai.chat.request"

	msg := EventMessage{
		EventType: "ai.chat.request",
		Timestamp: time.Now(),
		UserID:    &event.UserID,
		Data:      event,
	}

	return s.SendMessage(topic, msg)
}

// PublishKnowledgeProcessEvent 发布知识库处理事件
func (s *KafkaService) PublishKnowledgeProcessEvent(event KnowledgeProcessEvent) error {
	topic := fmt.Sprintf("knowledge.%s", event.Action)

	msg := EventMessage{
		EventType: fmt.Sprintf("knowledge.%s", event.Action),
		Timestamp: time.Now(),
		UserID:    &event.UserID,
		Data:      event,
	}

	return s.SendMessage(topic, msg)
}

// PublishOrderEvent 发布订单事件
func (s *KafkaService) PublishOrderEvent(orderID string, eventType string, userID uint, data interface{}) error {
	topic := fmt.Sprintf("order.%s", eventType)

	msg := EventMessage{
		EventType: fmt.Sprintf("order.%s", eventType),
		Timestamp: time.Now(),
		UserID:    &userID,
		Data:      data,
	}

	return s.SendMessage(topic, msg)
}

// PublishTokenEvent 发布Token事件
func (s *KafkaService) PublishTokenEvent(userID uint, eventType string, amount int64) error {
	topic := fmt.Sprintf("token.%s", eventType)

	msg := EventMessage{
		EventType: fmt.Sprintf("token.%s", eventType),
		Timestamp: time.Now(),
		UserID:    &userID,
		Data: map[string]interface{}{
			"user_id": userID,
			"amount":  amount,
		},
	}

	return s.SendMessage(topic, msg)
}

// SendBatch 批量发送消息
func (s *KafkaService) SendBatch(topic string, messages []interface{}) error {
	if s.producer == nil {
		return nil
	}

	for _, msg := range messages {
		if err := s.SendMessage(topic, msg); err != nil {
			logger.Error("Failed to send batch message", zap.Error(err))
			// 继续发送其他消息
		}
	}

	return nil
}
