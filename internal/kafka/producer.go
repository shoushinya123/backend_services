package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// Producer Kafka生产者
type Producer struct {
	producer sarama.SyncProducer
	topic    string
}

// GetProducerInstance 获取底层sarama producer实例（用于扩展功能）
func (p *Producer) GetProducerInstance() sarama.SyncProducer {
	return p.producer
}

// ConversationMessage 对话消息结构
type ConversationMessage struct {
	ConversationID string                 `json:"conversation_id"`
	UserID         uint                   `json:"user_id"`
	ModelID        uint                   `json:"model_id"`
	Message        MessageData            `json:"message"`
	ModelParams    map[string]interface{} `json:"model_params"`
	Usage          *UsageInfo             `json:"usage,omitempty"`
	Timestamp      time.Time              `json:"timestamp"`
}

// MessageData 消息数据
type MessageData struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// UsageInfo Token使用信息
type UsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

var globalProducer *Producer

// InitProducer 初始化Kafka生产者
func InitProducer(brokers []string, topic string) error {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Timeout = 10 * time.Second

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return fmt.Errorf("创建Kafka生产者失败: %w", err)
	}

	globalProducer = &Producer{
		producer: producer,
		topic:    topic,
	}

	logger.Info("Kafka生产者初始化成功", zap.Strings("brokers", brokers), zap.String("topic", topic))
	return nil
}

// GetProducer 获取全局生产者实例
func GetProducer() *Producer {
	return globalProducer
}

// SendMessage 发送消息到Kafka
func (p *Producer) SendMessage(msg *ConversationMessage) error {
	if p == nil || p.producer == nil {
		return fmt.Errorf("Kafka生产者未初始化")
	}

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 创建Kafka消息
	kafkaMsg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(fmt.Sprintf("%d-%s", msg.UserID, msg.ConversationID)),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("user_id"),
				Value: []byte(fmt.Sprintf("%d", msg.UserID)),
			},
			{
				Key:   []byte("model_id"),
				Value: []byte(fmt.Sprintf("%d", msg.ModelID)),
			},
		},
	}

	// 发送消息
	partition, offset, err := p.producer.SendMessage(kafkaMsg)
	if err != nil {
		logger.Error("发送Kafka消息失败", zap.Error(err))
		return fmt.Errorf("发送消息失败: %w", err)
	}

	logger.Debug("Kafka消息发送成功",
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
		zap.String("conversation_id", msg.ConversationID))

	return nil
}

// Close 关闭生产者
func (p *Producer) Close() error {
	if p != nil && p.producer != nil {
		return p.producer.Close()
	}
	return nil
}

// SendConversationMessage 发送对话消息（便捷方法）
func SendConversationMessage(conversationID string, userID, modelID uint, role, content string, modelParams map[string]interface{}, usage *UsageInfo) error {
	producer := GetProducer()
	if producer == nil {
		// 如果Kafka未配置，静默失败（不影响主流程）
		logger.Warn("Kafka生产者未初始化，跳过消息发送")
		return nil
	}

	msg := &ConversationMessage{
		ConversationID: conversationID,
		UserID:         userID,
		ModelID:        modelID,
		Message: MessageData{
			Role:      role,
			Content:   content,
			Timestamp: time.Now(),
		},
		ModelParams: modelParams,
		Usage:       usage,
		Timestamp:   time.Now(),
	}

	return producer.SendMessage(msg)
}
