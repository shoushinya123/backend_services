package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/aihub/backend-go/internal/kafka"
	"github.com/aihub/backend-go/internal/logger"
	"go.uber.org/zap"
)

// KnowledgeConsumer 知识库Kafka消费者服务
type KnowledgeConsumer struct {
	knowledgeService *KnowledgeService
	consumer         *kafka.Consumer
	maxRetries       int
}

// NewKnowledgeConsumer 创建知识库消费者
func NewKnowledgeConsumer(knowledgeService *KnowledgeService) *KnowledgeConsumer {
	return &KnowledgeConsumer{
		knowledgeService: knowledgeService,
		maxRetries:       3,
	}
}

// Start 启动消费者
func (kc *KnowledgeConsumer) Start() error {
	consumer := kafka.GetConsumer()
	if consumer == nil {
		return fmt.Errorf("Kafka消费者未初始化")
	}

	kc.consumer = consumer

	// 注册知识库处理事件处理器
	consumer.RegisterHandler("knowledge.process", kc.handleKnowledgeProcess)

	logger.Info("知识库Kafka消费者已启动")
	return nil
}

// handleKnowledgeProcess 处理知识库处理事件
func (kc *KnowledgeConsumer) handleKnowledgeProcess(ctx context.Context, message *sarama.ConsumerMessage) error {
	// 解析消息
	var event struct {
		EventType string                 `json:"event_type"`
		Timestamp time.Time              `json:"timestamp"`
		Data      map[string]interface{} `json:"data"`
	}

	if err := json.Unmarshal(message.Value, &event); err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 提取文档ID和知识库ID
	data := event.Data
	kbID, ok1 := data["knowledge_base_id"].(float64)
	docID, ok2 := data["document_id"].(float64)

	if !ok1 || !ok2 {
		return fmt.Errorf("消息格式错误：缺少必要字段")
	}

	documentID := uint(docID)
	knowledgeBaseID := uint(kbID)

	logger.Info("处理知识库文档",
		zap.Uint("knowledge_base_id", knowledgeBaseID),
		zap.Uint("document_id", documentID))

	// 处理文档
	if err := kc.knowledgeService.processDocument(documentID); err != nil {
		logger.Error("处理文档失败",
			zap.Uint("document_id", documentID),
			zap.Error(err))

		// 检查重试次数
		retryCount := 0
		if rc, ok := data["retry_count"].(float64); ok {
			retryCount = int(rc)
		}

		if retryCount < kc.maxRetries {
			// 发送重试消息
			retryCount++
			data["retry_count"] = retryCount
			retryData, _ := json.Marshal(event)

			if err := kafka.SendRetryMessage(
				message.Topic,
				string(message.Key),
				retryData,
				retryCount,
				kc.maxRetries,
				err.Error(),
			); err != nil {
				logger.Error("发送重试消息失败", zap.Error(err))
			} else {
				logger.Info("已发送重试消息",
					zap.Uint("document_id", documentID),
					zap.Int("retry_count", retryCount))
			}
		} else {
			// 超过最大重试次数，标记为失败
			logger.Error("文档处理失败，超过最大重试次数",
				zap.Uint("document_id", documentID),
				zap.Int("max_retries", kc.maxRetries))
		}

		return err
	}

	logger.Info("文档处理成功",
		zap.Uint("document_id", documentID))

	return nil
}

// Stop 停止消费者
func (kc *KnowledgeConsumer) Stop() error {
	if kc.consumer != nil {
		return kc.consumer.Close()
	}
	return nil
}

