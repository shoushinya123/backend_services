package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// TaskService 任务服务
type TaskService struct{}

// NewTaskService 创建任务服务实例
func NewTaskService() *TaskService {
	return &TaskService{}
}

// CreateCrawlerTaskRequest 创建爬虫任务请求
type CreateCrawlerTaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Config      map[string]interface{} `json:"config"`
}

// UpdateCrawlerTaskRequest 更新爬虫任务请求
type UpdateCrawlerTaskRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Config      map[string]interface{} `json:"config"`
}

// CreateProcessingTaskRequest 创建数据处理任务请求
type CreateProcessingTaskRequest struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	TaskType    string                   `json:"task_type"`
	InputFiles  []map[string]interface{} `json:"input_files"`
	Config      map[string]interface{}   `json:"config"`
}

// UpdateProcessingTaskRequest 更新数据处理任务请求
type UpdateProcessingTaskRequest struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	TaskType    string                   `json:"task_type"`
	InputFiles  []map[string]interface{} `json:"input_files"`
	Config      map[string]interface{}   `json:"config"`
}

// GetCrawlerTasks 获取爬虫任务列表
func (s *TaskService) GetCrawlerTasks(userID uint, page, limit int, status string) ([]models.CrawlerTask, int64, error) {
	var tasks []models.CrawlerTask
	var total int64

	offset := (page - 1) * limit

	query := database.DB.Model(&models.CrawlerTask{}).Where("created_by = ?", userID)

	// 添加状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	if err := query.Preload("Creator").
		Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// CreateCrawlerTask 创建爬虫任务
func (s *TaskService) CreateCrawlerTask(userID uint, req CreateCrawlerTaskRequest) (*models.CrawlerTask, error) {
	configJSON, _ := json.Marshal(req.Config)

	task := &models.CrawlerTask{
		Name:        req.Name,
		Description: req.Description,
		URL:         req.URL,
		Config:      string(configJSON),
		Status:      "pending",
		CreatedBy:   userID,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	if err := database.DB.Create(task).Error; err != nil {
		return nil, err
	}

	// 异步执行任务
	go s.executeCrawlerTask(task.TaskID)

	return task, nil
}

// UpdateCrawlerTask 更新爬虫任务
func (s *TaskService) UpdateCrawlerTask(taskID, userID uint, req UpdateCrawlerTaskRequest) (*models.CrawlerTask, error) {
	var task models.CrawlerTask
	if err := database.DB.Where("task_id = ? AND created_by = ?", taskID, userID).
		First(&task).Error; err != nil {
		return nil, err
	}

	// 只有待处理状态的任务可以更新
	if task.Status != "pending" {
		return nil, fmt.Errorf("只能更新待处理状态的任务")
	}

	configJSON, _ := json.Marshal(req.Config)

	task.Name = req.Name
	task.Description = req.Description
	task.URL = req.URL
	task.Config = string(configJSON)
	task.UpdateTime = time.Now()

	if err := database.DB.Save(&task).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

// DeleteCrawlerTask 删除爬虫任务
func (s *TaskService) DeleteCrawlerTask(taskID, userID uint) error {
	var task models.CrawlerTask
	if err := database.DB.Where("task_id = ? AND created_by = ?", taskID, userID).
		First(&task).Error; err != nil {
		return err
	}

	// 只有非运行状态的任务可以删除
	if task.Status == "running" {
		return fmt.Errorf("运行中的任务不能删除")
	}

	return database.DB.Delete(&task).Error
}

// RunCrawlerTask 执行爬虫任务
func (s *TaskService) RunCrawlerTask(taskID, userID uint) error {
	var task models.CrawlerTask
	if err := database.DB.Where("task_id = ? AND created_by = ?", taskID, userID).
		First(&task).Error; err != nil {
		return err
	}

	if task.Status == "running" {
		return fmt.Errorf("任务正在运行中")
	}

	// 更新任务状态
	task.Status = "running"
	now := time.Now()
	task.StartedAt = &now
	task.UpdateTime = time.Now()

	if err := database.DB.Save(&task).Error; err != nil {
		return err
	}

	// 异步执行任务
	go s.executeCrawlerTask(taskID)

	return nil
}

// GetProcessingTasks 获取数据处理任务列表
func (s *TaskService) GetProcessingTasks(userID uint, page, limit int, status string) ([]models.ProcessingTask, int64, error) {
	var tasks []models.ProcessingTask
	var total int64

	offset := (page - 1) * limit

	query := database.DB.Model(&models.ProcessingTask{}).Where("created_by = ?", userID)

	// 添加状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	if err := query.Preload("Creator").
		Order("create_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// CreateProcessingTask 创建数据处理任务
func (s *TaskService) CreateProcessingTask(userID uint, req CreateProcessingTaskRequest) (*models.ProcessingTask, error) {
	inputFilesJSON, _ := json.Marshal(req.InputFiles)
	configJSON, _ := json.Marshal(req.Config)

	task := &models.ProcessingTask{
		Name:        req.Name,
		Description: req.Description,
		TaskType:    req.TaskType,
		InputFiles:  string(inputFilesJSON),
		Config:      string(configJSON),
		Status:      "pending",
		CreatedBy:   userID,
		CreateTime:  time.Now(),
		UpdateTime:  time.Now(),
	}

	if err := database.DB.Create(task).Error; err != nil {
		return nil, err
	}

	// 异步执行任务
	go s.executeProcessingTask(task.TaskID)

	return task, nil
}

// UpdateProcessingTask 更新数据处理任务
func (s *TaskService) UpdateProcessingTask(taskID, userID uint, req UpdateProcessingTaskRequest) (*models.ProcessingTask, error) {
	var task models.ProcessingTask
	if err := database.DB.Where("task_id = ? AND created_by = ?", taskID, userID).
		First(&task).Error; err != nil {
		return nil, err
	}

	// 只有待处理状态的任务可以更新
	if task.Status != "pending" {
		return nil, fmt.Errorf("只能更新待处理状态的任务")
	}

	inputFilesJSON, _ := json.Marshal(req.InputFiles)
	configJSON, _ := json.Marshal(req.Config)

	task.Name = req.Name
	task.Description = req.Description
	task.TaskType = req.TaskType
	task.InputFiles = string(inputFilesJSON)
	task.Config = string(configJSON)
	task.UpdateTime = time.Now()

	if err := database.DB.Save(&task).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

// DeleteProcessingTask 删除数据处理任务
func (s *TaskService) DeleteProcessingTask(taskID, userID uint) error {
	var task models.ProcessingTask
	if err := database.DB.Where("task_id = ? AND created_by = ?", taskID, userID).
		First(&task).Error; err != nil {
		return err
	}

	// 只有非运行状态的任务可以删除
	if task.Status == "running" {
		return fmt.Errorf("运行中的任务不能删除")
	}

	return database.DB.Delete(&task).Error
}

// RunProcessingTask 执行数据处理任务
func (s *TaskService) RunProcessingTask(taskID, userID uint) error {
	var task models.ProcessingTask
	if err := database.DB.Where("task_id = ? AND created_by = ?", taskID, userID).
		First(&task).Error; err != nil {
		return err
	}

	if task.Status == "running" {
		return fmt.Errorf("任务正在运行中")
	}

	// 更新任务状态
	task.Status = "running"
	now := time.Now()
	task.StartedAt = &now
	task.UpdateTime = time.Now()

	if err := database.DB.Save(&task).Error; err != nil {
		return err
	}

	// 异步执行任务
	go s.executeProcessingTask(taskID)

	return nil
}

// executeCrawlerTask 执行爬虫任务（模拟实现）
func (s *TaskService) executeCrawlerTask(taskID uint) {
	// 更新任务状态为运行中
	now := time.Now()
	database.DB.Model(&models.CrawlerTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":     "running",
		"started_at": &now,
	})

	// 模拟任务执行
	time.Sleep(5 * time.Second) // 模拟5秒的执行时间

	// 更新任务结果
	result := map[string]interface{}{
		"url":         "https://example.com",
		"status_code": 200,
		"content_length": 1024,
		"extracted_data": []string{"数据1", "数据2", "数据3"},
	}
	resultJSON, _ := json.Marshal(result)

	completedAt := time.Now()
	database.DB.Model(&models.CrawlerTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":        "completed",
		"result":        string(resultJSON),
		"completed_at":  &completedAt,
		"update_time":   completedAt,
	})
}

// executeProcessingTask 执行数据处理任务（模拟实现）
func (s *TaskService) executeProcessingTask(taskID uint) {
	// 更新任务状态为运行中
	now := time.Now()
	database.DB.Model(&models.ProcessingTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":     "running",
		"started_at": &now,
	})

	// 模拟任务执行
	time.Sleep(3 * time.Second) // 模拟3秒的执行时间

	// 更新任务结果
	result := map[string]interface{}{
		"processed_files": 2,
		"output_files":    []string{"output1.txt", "output2.txt"},
		"statistics": map[string]interface{}{
			"total_lines": 1000,
			"processed_lines": 950,
			"errors": 5,
		},
	}
	resultJSON, _ := json.Marshal(result)

	outputFiles := []string{"processed_output1.txt", "processed_output2.txt"}
	outputFilesJSON, _ := json.Marshal(outputFiles)

	completedAt := time.Now()
	database.DB.Model(&models.ProcessingTask{}).Where("task_id = ?", taskID).Updates(map[string]interface{}{
		"status":        "completed",
		"result":        string(resultJSON),
		"output_files":  string(outputFilesJSON),
		"completed_at":  &completedAt,
		"update_time":   completedAt,
	})
}
