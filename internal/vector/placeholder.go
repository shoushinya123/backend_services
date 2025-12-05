package vector

// VectorDB 向量数据库接口
type VectorDB interface {
	Search(query string, limit int) ([]Result, error)
	VectorizeMessage(ctx interface{}, messageID interface{}, content string) ([]float32, error)
	IsConfigured() bool
}

// Result 搜索结果
type Result struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// Message 消息
type Message struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// GetVectorDB 获取向量数据库实例
func GetVectorDB() VectorDB {
	return nil
}
