package middleware

import (
	"encoding/json"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
	"github.com/beego/beego/v2/server/web/context"
)

// AuditLogFilter 操作审计过滤器
// operationType: 操作类型（如 "CREATE", "UPDATE", "DELETE", "VIEW"）
// resourceType: 资源类型（如 "user", "order", "model"）
func AuditLogFilter(operationType, resourceType string) func(*context.Context) {
	return func(ctx *context.Context) {
		// 在请求处理完成后记录日志
		ctx.Input.SetData("auditOperationType", operationType)
		ctx.Input.SetData("auditResourceType", resourceType)
	}
}

// RecordAuditLog 记录操作审计日志（在控制器中调用）
func RecordAuditLog(ctx *context.Context, userID uint, operationType, resourceType, resourceID, action string, detail map[string]interface{}, status string, errorMsg string) {
	// 获取IP地址和User-Agent
	ipAddress := ctx.Request.RemoteAddr
	if forwarded := ctx.Request.Header.Get("X-Forwarded-For"); forwarded != "" {
		ipAddress = forwarded
	}
	userAgent := ctx.Request.Header.Get("User-Agent")

	// 序列化详情
	detailJSON := "{}"
	if detail != nil {
		if bytes, err := json.Marshal(detail); err == nil {
			detailJSON = string(bytes)
		}
	}

	// 创建操作日志
	log := models.OperationLog{
		UserID:        userID,
		OperationType: operationType,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Action:        action,
		Detail:        detailJSON,
		IPAddress:     ipAddress,
		UserAgent:     userAgent,
		Status:        status,
		ErrorMessage:  errorMsg,
		CreateTime:    time.Now(),
	}

	// 异步保存日志（不阻塞请求）
	go func() {
		if err := database.DB.Create(&log).Error; err != nil {
			// 记录失败不影响主流程，可以记录到日志系统
			// logger.Error("记录操作日志失败", zap.Error(err))
		}
	}()
}

// RecordAuditLogFromContext 从上下文记录操作日志
func RecordAuditLogFromContext(ctx *context.Context, userID uint, resourceID string, action string, detail map[string]interface{}, status string, errorMsg string) {
	operationType := ctx.Input.GetData("auditOperationType")
	resourceType := ctx.Input.GetData("auditResourceType")

	opType := "UNKNOWN"
	resType := "UNKNOWN"

	if ot, ok := operationType.(string); ok {
		opType = ot
	}
	if rt, ok := resourceType.(string); ok {
		resType = rt
	}

	RecordAuditLog(ctx, userID, opType, resType, resourceID, action, detail, status, errorMsg)
}

