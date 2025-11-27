package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// Consumer Kafka消费者
type Consumer struct {
	consumer sarama.ConsumerGroup
	groupID  string
	topics   []string
	handlers map[string]MessageHandler
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// MessageHandler 消息处理函数
type MessageHandler func(ctx context.Context, message *sarama.ConsumerMessage) error

var globalConsumer *Consumer

// InitConsumer 初始化Kafka消费者
func InitConsumer(brokers []string, groupID string, topics []string) error {
	config := sarama.NewConfig()
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true
	config.Version = sarama.V2_6_0_0

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return fmt.Errorf("创建Kafka消费者组失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	globalConsumer = &Consumer{
		consumer: consumerGroup,
		groupID:  groupID,
		topics:   topics,
		handlers: make(map[string]MessageHandler),
		ctx:      ctx,
		cancel:   cancel,
	}

	logger.Info("Kafka消费者初始化成功",
		zap.Strings("brokers", brokers),
		zap.String("group_id", groupID),
		zap.Strings("topics", topics))

	// 启动消费者
	go globalConsumer.start()

	return nil
}

// GetConsumer 获取全局消费者实例
func GetConsumer() *Consumer {
	return globalConsumer
}

// RegisterHandler 注册消息处理器
func (c *Consumer) RegisterHandler(topic string, handler MessageHandler) {
	if c == nil {
		return
	}
	c.handlers[topic] = handler
	logger.Info("注册Kafka消息处理器", zap.String("topic", topic))
}

// start 启动消费者
func (c *Consumer) start() {
	if c == nil || c.consumer == nil {
		return
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				logger.Info("Kafka消费者停止")
				return
			default:
				// 消费消息
				handler := &consumerGroupHandler{
					handlers: c.handlers,
				}
				err := c.consumer.Consume(c.ctx, c.topics, handler)
				if err != nil {
					logger.Error("消费消息失败", zap.Error(err))
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()

	// 处理错误
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for err := range c.consumer.Errors() {
			logger.Error("Kafka消费者错误", zap.Error(err))
		}
	}()
}

// Close 关闭消费者
func (c *Consumer) Close() error {
	if c == nil {
		return nil
	}
	c.cancel()
	c.wg.Wait()
	if c.consumer != nil {
		return c.consumer.Close()
	}
	return nil
}

// consumerGroupHandler 消费者组处理器
type consumerGroupHandler struct {
	handlers map[string]MessageHandler
}

// Setup 会话开始
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 会话结束
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// 查找处理器
			handler, ok := h.handlers[message.Topic]
			if !ok {
				logger.Warn("未找到消息处理器", zap.String("topic", message.Topic))
				session.MarkMessage(message, "")
				continue
			}

			// 处理消息
			ctx := context.Background()
			if err := handler(ctx, message); err != nil {
				logger.Error("处理消息失败",
					zap.String("topic", message.Topic),
					zap.Int("partition", int(message.Partition)),
					zap.Int64("offset", message.Offset),
					zap.Error(err))
				// 不标记消息，等待重试
				continue
			}

			// 标记消息已处理
			session.MarkMessage(message, "")
			logger.Debug("消息处理成功",
				zap.String("topic", message.Topic),
				zap.Int("partition", int(message.Partition)),
				zap.Int64("offset", message.Offset))

		case <-session.Context().Done():
			return nil
		}
	}
}

// KnowledgeProcessMessage 知识库处理消息
type KnowledgeProcessMessage struct {
	KnowledgeBaseID uint   `json:"knowledge_base_id"`
	DocumentID      uint   `json:"document_id,omitempty"`
	Action          string `json:"action"`
	UserID          uint   `json:"user_id"`
	RetryCount      int    `json:"retry_count,omitempty"`
}

// ParseKnowledgeProcessMessage 解析知识库处理消息
func ParseKnowledgeProcessMessage(data []byte) (*KnowledgeProcessMessage, error) {
	var msg KnowledgeProcessMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("解析消息失败: %w", err)
	}
	return &msg, nil
}

// RetryMessage 重试消息
type RetryMessage struct {
	OriginalTopic string          `json:"original_topic"`
	OriginalKey   string          `json:"original_key"`
	Data          json.RawMessage `json:"data"`
	RetryCount    int             `json:"retry_count"`
	MaxRetries    int             `json:"max_retries"`
	LastError     string          `json:"last_error,omitempty"`
}

// SendRetryMessage 发送重试消息
func SendRetryMessage(topic string, key string, data []byte, retryCount, maxRetries int, lastError string) error {
	retryMsg := RetryMessage{
		OriginalTopic: topic,
		OriginalKey:   key,
		Data:          data,
		RetryCount:    retryCount,
		MaxRetries:    maxRetries,
		LastError:     lastError,
	}

	retryData, err := json.Marshal(retryMsg)
	if err != nil {
		return fmt.Errorf("序列化重试消息失败: %w", err)
	}

	producer := GetProducer()
	if producer == nil {
		return fmt.Errorf("Kafka生产者未初始化")
	}

	saramaProducer := producer.GetProducerInstance()
	if saramaProducer == nil {
		return fmt.Errorf("Sarama生产者未初始化")
	}

	kafkaMsg := &sarama.ProducerMessage{
		Topic: fmt.Sprintf("%s.retry", topic),
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(retryData),
	}

	_, _, err = saramaProducer.SendMessage(kafkaMsg)
	return err
}
