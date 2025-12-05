package workflow

// NodeMetadata 节点元数据
type NodeMetadata struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
}

// NodeFactory 节点工厂
type NodeFactory struct{}
