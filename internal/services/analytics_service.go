package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/aihub/backend-go/internal/database"
	"github.com/aihub/backend-go/internal/models"
)

// AnalyticsService 统计分析服务
type AnalyticsService struct{}

// NewAnalyticsService 创建统计分析服务实例
func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

// OverviewData 概览数据
type OverviewData struct {
	TotalUsers       int64     `json:"total_users"`
	ActiveUsers      int64     `json:"active_users"`
	TotalInputTokens int64     `json:"total_input_tokens"`
	TotalOutputTokens int64    `json:"total_output_tokens"`
	TotalRevenue     int64     `json:"total_revenue"`
	Period           Period    `json:"period"`
}

// Period 时间段
type Period struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// TrendData 趋势数据
type TrendData struct {
	Date        string `json:"date"`
	InputTokens int64  `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
	Revenue     int64  `json:"revenue"`
}

// ModelDistribution 模型分布数据
type ModelDistribution struct {
	ModelName  string `json:"model_name"`
	TotalTokens int64 `json:"total_tokens"`
	Revenue     int64 `json:"revenue"`
	CallCount   int64 `json:"call_count"`
}

// Anomaly 异常数据
type Anomaly struct {
	UserID     uint   `json:"user_id"`
	Username   string `json:"username"`
	TotalTokens int64 `json:"total_tokens"`
	Threshold  float64 `json:"threshold"`
	Severity   string `json:"severity"`
}

// GetOverview 获取仪表盘概览数据
func (s *AnalyticsService) GetOverview(startDate, endDate time.Time) (*OverviewData, error) {
	cacheKey := fmt.Sprintf("overview:%s:%s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// 尝试从缓存获取
	if database.RedisClient != nil {
		ctx := context.Background()
		cached, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var data OverviewData
			if err := json.Unmarshal([]byte(cached), &data); err == nil {
				return &data, nil
			}
		}
	}

	// 总用户数
	var totalUsers int64
	database.DB.Model(&models.User{}).Count(&totalUsers)

	// Token消耗统计
	var tokenStats struct {
		TotalInput  int64
		TotalOutput int64
		TotalRevenue int64
	}
	database.DB.Model(&models.BillingRecord{}).
		Where("create_time >= ? AND create_time <= ?", startDate, endDate).
		Select("COALESCE(SUM(input_tokens), 0) as total_input, "+
			"COALESCE(SUM(output_tokens), 0) as total_output, "+
			"COALESCE(SUM(amount), 0) as total_revenue").
		Scan(&tokenStats)

	// 活跃用户数（有计费记录的用户）
	var activeUsers int64
	database.DB.Model(&models.BillingRecord{}).
		Where("create_time >= ? AND create_time <= ?", startDate, endDate).
		Distinct("user_id").
		Count(&activeUsers)

	result := &OverviewData{
		TotalUsers:       totalUsers,
		ActiveUsers:      activeUsers,
		TotalInputTokens: tokenStats.TotalInput,
		TotalOutputTokens: tokenStats.TotalOutput,
		TotalRevenue:     tokenStats.TotalRevenue,
		Period: Period{
			Start: startDate.Format("2006-01-02"),
			End:   endDate.Format("2006-01-02"),
		},
	}

	// 缓存结果（1小时）
	if database.RedisClient != nil {
		ctx := context.Background()
		if dataJSON, err := json.Marshal(result); err == nil {
			database.RedisClient.SetEx(ctx, cacheKey, string(dataJSON), time.Hour)
		}
	}

	return result, nil
}

// GetTokenTrend 获取Token消耗趋势
func (s *AnalyticsService) GetTokenTrend(startDate, endDate time.Time, groupBy string) ([]TrendData, error) {
	cacheKey := fmt.Sprintf("token_trend:%s:%s:%s",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), groupBy)

	// 尝试从缓存获取
	if database.RedisClient != nil {
		ctx := context.Background()
		cached, err := database.RedisClient.Get(ctx, cacheKey).Result()
		if err == nil && cached != "" {
			var data []TrendData
			if err := json.Unmarshal([]byte(cached), &data); err == nil {
				return data, nil
			}
		}
	}

	// 根据分组方式确定时间格式
	var dateFormat string
	var dateTrunc string
	switch groupBy {
	case "week":
		dateFormat = "2006-01"
		dateTrunc = "week"
	case "month":
		dateFormat = "2006-01"
		dateTrunc = "month"
	default: // day
		dateFormat = "2006-01-02"
		dateTrunc = "day"
	}

	// 查询数据
	var results []struct {
		Date        time.Time `gorm:"column:date"`
		InputTokens int64     `gorm:"column:input_tokens"`
		OutputTokens int64   `gorm:"column:output_tokens"`
		Revenue     int64     `gorm:"column:revenue"`
	}

	// 构建SQL查询
	sql := fmt.Sprintf(`SELECT 
		DATE_TRUNC('%s', create_time) as date,
		COALESCE(SUM(input_tokens), 0) as input_tokens,
		COALESCE(SUM(output_tokens), 0) as output_tokens,
		COALESCE(SUM(amount), 0) as revenue
	FROM billing_record
	WHERE create_time >= ? AND create_time <= ?
	GROUP BY date
	ORDER BY date`, dateTrunc)

	err := database.DB.Raw(sql, startDate, endDate).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	trendData := make([]TrendData, 0, len(results))
	for _, r := range results {
		trendData = append(trendData, TrendData{
			Date:        r.Date.Format(dateFormat),
			InputTokens: r.InputTokens,
			OutputTokens: r.OutputTokens,
			Revenue:     r.Revenue,
		})
	}

	// 缓存结果（1小时）
	if database.RedisClient != nil {
		ctx := context.Background()
		if dataJSON, err := json.Marshal(trendData); err == nil {
			database.RedisClient.SetEx(ctx, cacheKey, string(dataJSON), time.Hour)
		}
	}

	return trendData, nil
}

// GetModelDistribution 获取模型使用分布
func (s *AnalyticsService) GetModelDistribution(startDate, endDate time.Time) ([]ModelDistribution, error) {
	var results []struct {
		ModelName   string
		TotalTokens int64
		Revenue     int64
		CallCount   int64
	}

	err := database.DB.Model(&models.BillingRecord{}).
		Select("m.name as model_name, "+
			"COALESCE(SUM(br.input_tokens + br.output_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(br.amount), 0) as revenue, "+
			"COUNT(br.record_id) as call_count").
		Joins("JOIN model m ON br.model_id = m.model_id").
		Where("br.create_time >= ? AND br.create_time <= ?", startDate, endDate).
		Group("m.model_id, m.name").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	distributions := make([]ModelDistribution, 0, len(results))
	for _, r := range results {
		distributions = append(distributions, ModelDistribution{
			ModelName:   r.ModelName,
			TotalTokens: r.TotalTokens,
			Revenue:     r.Revenue,
			CallCount:   r.CallCount,
		})
	}

	return distributions, nil
}

// DetectAnomalies 异常检测
func (s *AnalyticsService) DetectAnomalies(threshold float64) ([]Anomaly, error) {
	// 获取最近7天的数据
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	// 按用户统计token消耗
	var userStats []struct {
		UserID      uint
		TotalTokens int64
	}

	err := database.DB.Model(&models.BillingRecord{}).
		Select("user_id, COALESCE(SUM(input_tokens + output_tokens), 0) as total_tokens").
		Where("create_time >= ? AND create_time <= ?", startDate, endDate).
		Group("user_id").
		Scan(&userStats).Error

	if err != nil {
		return nil, err
	}

	if len(userStats) == 0 {
		return []Anomaly{}, nil
	}

	// 计算平均值和标准差
	tokensList := make([]float64, len(userStats))
	for i, stat := range userStats {
		tokensList[i] = float64(stat.TotalTokens)
	}

	mean := 0.0
	for _, v := range tokensList {
		mean += v
	}
	mean /= float64(len(tokensList))

	variance := 0.0
	for _, v := range tokensList {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(tokensList))
	stdDev := math.Sqrt(variance)

	// 找出异常用户
	anomalies := make([]Anomaly, 0)
	thresholdValue := mean + threshold*stdDev

	for _, stat := range userStats {
		if float64(stat.TotalTokens) > thresholdValue {
			var user models.User
			if err := database.DB.First(&user, stat.UserID).Error; err != nil {
				continue
			}

			severity := "medium"
			if float64(stat.TotalTokens) > mean+5*stdDev {
				severity = "high"
			}

			anomalies = append(anomalies, Anomaly{
				UserID:      stat.UserID,
				Username:    user.Username,
				TotalTokens: stat.TotalTokens,
				Threshold:   thresholdValue,
				Severity:    severity,
			})
		}
	}

	// 按总token数降序排序
	for i := 0; i < len(anomalies)-1; i++ {
		for j := i + 1; j < len(anomalies); j++ {
			if anomalies[i].TotalTokens < anomalies[j].TotalTokens {
				anomalies[i], anomalies[j] = anomalies[j], anomalies[i]
			}
		}
	}

	return anomalies, nil
}

