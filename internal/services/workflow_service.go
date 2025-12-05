//go:build !knowledge
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
	"github.com/aihub/backend-go/internal/workflow"
)

// WorkflowService 工作流服务
type WorkflowService struct{}

// NewWorkflowService 创建工作流服务实例
func NewWorkflowService() *WorkflowService {
	return &WorkflowService{}
}

// CreateWorkflowRequest 创建工作流请求
type CreateWorkflowRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Config      interface{} `json:"config"`
}

// UpdateWorkflowRequest 更新工作流请求
type UpdateWorkflowRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Config      interface{} `json:"config"`
}

// GetWorkflows 获取工作流列表
func (s *WorkflowService) GetWorkflows(userID uint, page, limit int, status string) ([]models.Workflow, int64, error) {
	var workflows []models.Workflow
	var total int64

	offset := (page - 1) * limit

	query := database.DB.Model(&models.Workflow{}).Where("owner_id = ?", userID)

	// 添加状态筛选
	if status != "" {
		if status == "active" {
			query = query.Where("is_active = ?", true)
		} else if status == "inactive" {
			query = query.Where("is_active = ?", false)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	if err := query.Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&workflows).Error; err != nil {
		return nil, 0, err
	}

	return workflows, total, nil
}

// GetWorkflow 获取单个工作流
func (s *WorkflowService) GetWorkflow(workflowID, userID uint) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := database.DB.Where("workflow_id = ? AND owner_id = ?", workflowID, userID).
		First(&workflow).Error; err != nil {
		return nil, err
	}
	return &workflow, nil
}

// CreateWorkflow 创建工作流
func (s *WorkflowService) CreateWorkflow(userID uint, req CreateWorkflowRequest) (*models.Workflow, error) {
	configJSON, _ := json.Marshal(req.Config)

	workflow := &models.Workflow{
		Name:        req.Name,
		Description: req.Description,
		Config:      string(configJSON),
		OwnerID:     userID,
		IsActive:    true,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	if err := database.DB.Create(workflow).Error; err != nil {
		return nil, err
	}

	return workflow, nil
}

// UpdateWorkflow 更新工作流
func (s *WorkflowService) UpdateWorkflow(workflowID, userID uint, req UpdateWorkflowRequest) (*models.Workflow, error) {
	var workflow models.Workflow
	if err := database.DB.Where("workflow_id = ? AND owner_id = ?", workflowID, userID).
		First(&workflow).Error; err != nil {
		return nil, err
	}

	configJSON, _ := json.Marshal(req.Config)

	workflow.Name = req.Name
	workflow.Description = req.Description
	workflow.Config = string(configJSON)
	workflow.UpdateTime = time.Now()

	if err := database.DB.Save(&workflow).Error; err != nil {
		return nil, err
	}

	return &workflow, nil
}

// DeleteWorkflow 删除工作流
func (s *WorkflowService) DeleteWorkflow(workflowID, userID uint) error {
	var workflow models.Workflow
	if err := database.DB.Where("workflow_id = ? AND owner_id = ?", workflowID, userID).
		First(&workflow).Error; err != nil {
		return err
	}

	return database.DB.Delete(&workflow).Error
}

// RunWorkflow 运行工作流
func (s *WorkflowService) RunWorkflow(workflowID, userID uint) error {
	var workflow models.Workflow
	if err := database.DB.Where("workflow_id = ? AND owner_id = ?", workflowID, userID).
		First(&workflow).Error; err != nil {
		return err
	}

	if !workflow.IsActive {
		return fmt.Errorf("工作流未激活")
	}

	// 异步执行工作流
	go s.executeWorkflow(workflowID)

	return nil
}

// PauseWorkflow 暂停工作流
func (s *WorkflowService) PauseWorkflow(workflowID, userID uint) error {
	var workflow models.Workflow
	if err := database.DB.Where("workflow_id = ? AND owner_id = ?", workflowID, userID).
		First(&workflow).Error; err != nil {
		return err
	}

	// 这里应该更新工作流的执行状态
	// 现在只是简单地记录操作
	fmt.Printf("暂停工作流: %d\n", workflowID)

	return nil
}

// StopWorkflow 停止工作流
func (s *WorkflowService) StopWorkflow(workflowID, userID uint) error {
	var workflow models.Workflow
	if err := database.DB.Where("workflow_id = ? AND owner_id = ?", workflowID, userID).
		First(&workflow).Error; err != nil {
		return err
	}

	// 这里应该停止工作流的执行
	// 现在只是简单地记录操作
	fmt.Printf("停止工作流: %d\n", workflowID)

	return nil
}

// GetWorkflowTemplates 获取工作流模板
func (s *WorkflowService) GetWorkflowTemplates() ([]map[string]interface{}, error) {
	// 返回预定义的工作流模板
	templates := []map[string]interface{}{
		{
			"id":          "text_analysis",
			"name":        "文本分析工作流",
			"description": "用于分析和处理文本数据的自动化工作流",
			"nodes": []map[string]interface{}{
				{
					"type": "input",
					"name": "文本输入",
					"config": map[string]interface{}{
						"input_type": "text",
					},
				},
				{
					"type": "llm",
					"name": "AI分析",
					"config": map[string]interface{}{
						"model": "gpt-4",
						"prompt": "请分析以下文本的内容和情感倾向",
					},
				},
				{
					"type": "output",
					"name": "结果输出",
					"config": map[string]interface{}{
						"output_format": "json",
					},
				},
			},
		},
		{
			"id":          "data_processing",
			"name":        "数据处理工作流",
			"description": "用于批量数据处理和转换的工作流",
			"nodes": []map[string]interface{}{
				{
					"type": "input",
					"name": "数据输入",
					"config": map[string]interface{}{
						"input_type": "file",
					},
				},
				{
					"type": "transform",
					"name": "数据转换",
					"config": map[string]interface{}{
						"operation": "clean",
					},
				},
				{
					"type": "output",
					"name": "处理结果",
					"config": map[string]interface{}{
						"output_format": "csv",
					},
				},
			},
		},
		{
			"id":          "ai_assistant",
			"name":        "AI助手工作流",
			"description": "智能问答和助手服务工作流",
			"nodes": []map[string]interface{}{
				{
					"type": "input",
					"name": "用户查询",
					"config": map[string]interface{}{
						"input_type": "text",
					},
				},
				{
					"type": "knowledge_search",
					"name": "知识库搜索",
					"config": map[string]interface{}{
						"knowledge_base_id": 1,
					},
				},
				{
					"type": "llm",
					"name": "智能回复",
					"config": map[string]interface{}{
						"model": "gpt-4",
						"system_prompt": "你是一个专业的AI助手，请基于提供的知识回答用户问题",
					},
				},
				{
					"type": "output",
					"name": "回复输出",
					"config": map[string]interface{}{
						"output_format": "text",
					},
				},
			},
		},
	}

	return templates, nil
}

// executeWorkflow 执行工作流
func (s *WorkflowService) executeWorkflow(workflowID uint) {
	var wf models.Workflow
	if err := database.DB.First(&wf, workflowID).Error; err != nil {
		fmt.Printf("获取工作流失败: %v\n", err)
		return
	}

	// 解析工作流配置
	var config workflow.WorkflowConfig
	if err := json.Unmarshal([]byte(wf.Config), &config); err != nil {
		fmt.Printf("解析工作流配置失败: %v\n", err)
		return
	}

	// 创建执行记录
	executionID := uint(time.Now().Unix())
	execution := &models.WorkflowExecution{
		ExecutionID: executionID,
		WorkflowID:  workflowID,
		Status:      "running",
		TriggerType: "manual",
		OperatorID:  wf.OwnerID,
		StartTime:   time.Now(),
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}
	database.DB.Create(execution)

	// 执行工作流
	executor := workflow.NewWorkflowExecutor()
	ctx := context.Background()
	execCtx, err := executor.ExecuteWorkflow(ctx, config, fmt.Sprintf("%d", executionID), workflowID, wf.OwnerID)
	
	// 更新执行记录
	execution.Status = "completed"
	if err != nil {
		execution.Status = "failed"
		execution.Error = err.Error()
	}
	
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = int64(endTime.Sub(execution.StartTime).Milliseconds())
	
	logsJSON, _ := json.Marshal(execCtx.Logs)
	execution.Logs = string(logsJSON)
	
	resultJSON, _ := json.Marshal(execCtx.Variables)
	execution.Result = string(resultJSON)
	
	execution.UpdateTime = time.Now()
	database.DB.Save(execution)
	
	// 保存节点执行记录
	for nodeID, nodeResult := range execCtx.Results {
		nodeExec := &models.NodeExecution{
			ExecutionID:        executionID,
			NodeID:             nodeID,
			NodeType:           "unknown", // TODO: 从节点配置中获取
			Status:             "completed",
			WorkflowExecutionID: executionID,
			StartTime:          nodeResult.Timestamp,
			CreateTime:         time.Now(),
			UpdateTime:         time.Now(),
		}
		
		if !nodeResult.Success {
			nodeExec.Status = "failed"
			nodeExec.Error = nodeResult.Error
		}
		
		endTime := nodeResult.Timestamp.Add(nodeResult.Duration)
		nodeExec.EndTime = &endTime
		nodeExec.Duration = int64(nodeResult.Duration.Milliseconds())
		
		outputJSON, _ := json.Marshal(nodeResult.Output)
		nodeExec.Output = string(outputJSON)
		
		database.DB.Create(nodeExec)
	}
}

// GetNodeMetadata 获取节点元数据
func (s *WorkflowService) GetNodeMetadata() []workflow.NodeMetadata {
	factory := &workflow.NodeFactory{}
	return factory.GetAllNodeMetadata()
}
