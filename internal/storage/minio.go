package storage

import (
	"errors"
)

// InitMinIO initializes MinIO client (optional, can be nil if not configured)
func InitMinIO() (interface{}, error) {
	// MinIO is optional, return nil if not configured
	// This allows the application to start without MinIO
	return nil, errors.New("MinIO not configured, using local storage")
}








