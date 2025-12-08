package plugins

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strings"
)

// PluginLoader 插件加载器
type PluginLoader struct {
	pluginDir string
	tempDir   string
}

// NewPluginLoader 创建插件加载器
func NewPluginLoader(pluginDir, tempDir string) *PluginLoader {
	return &PluginLoader{
		pluginDir: pluginDir,
		tempDir:   tempDir,
	}
}

// LoadPlugin 加载xpkg插件包
func (l *PluginLoader) LoadPlugin(xpkgPath string) (Plugin, error) {
	// 1. 解压xpkg文件
	extractDir, err := l.extractXpkg(xpkgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract xpkg: %w", err)
	}
	defer os.RemoveAll(extractDir)

	// 2. 读取manifest.json
	manifestPath := filepath.Join(extractDir, "manifest.json")
	metadata, err := LoadMetadataFromManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %w", err)
	}

	// 3. 验证校验和（如果提供）
	if metadata.Checksum != "" {
		if err := l.verifyChecksum(xpkgPath, metadata.Checksum); err != nil {
			return nil, fmt.Errorf("checksum verification failed: %w", err)
		}
	}

	// 4. 加载插件二进制
	pluginPath := filepath.Join(extractDir, "plugin.so")
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		// 尝试wasm格式
		pluginPath = filepath.Join(extractDir, "plugin.wasm")
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("plugin binary not found (plugin.so or plugin.wasm)")
		}
		// TODO: 支持WASM加载
		return nil, fmt.Errorf("WASM plugin support not implemented yet")
	}

	// 5. 加载Go plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// 6. 查找插件符号
	sym, err := p.Lookup("NewPlugin")
	if err != nil {
		return nil, fmt.Errorf("plugin symbol 'NewPlugin' not found: %w", err)
	}

	// 7. 调用插件构造函数
	newPlugin, ok := sym.(func() Plugin)
	if !ok {
		return nil, fmt.Errorf("plugin symbol 'NewPlugin' has wrong type")
	}

	pluginInstance := newPlugin()

	// 8. 验证插件元数据
	if pluginInstance.Metadata().ID != metadata.ID {
		return nil, fmt.Errorf("plugin ID mismatch: manifest=%s, plugin=%s", metadata.ID, pluginInstance.Metadata().ID)
	}

	return pluginInstance, nil
}

// extractXpkg 解压xpkg文件
func (l *PluginLoader) extractXpkg(xpkgPath string) (string, error) {
	// 创建临时目录
	extractDir := filepath.Join(l.tempDir, fmt.Sprintf("plugin_%s", filepath.Base(xpkgPath)))
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// 打开ZIP文件
	r, err := zip.OpenReader(xpkgPath)
	if err != nil {
		return "", fmt.Errorf("failed to open xpkg: %w", err)
	}
	defer r.Close()

	// 解压所有文件
	for _, f := range r.File {
		path := filepath.Join(extractDir, f.Name)

		// 安全检查：防止路径遍历攻击
		if !strings.HasPrefix(path, extractDir) {
			return "", fmt.Errorf("invalid file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.FileInfo().Mode())
			continue
		}

		// 创建文件
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", fmt.Errorf("failed to create dir: %w", err)
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.FileInfo().Mode())
		if err != nil {
			return "", fmt.Errorf("failed to create file: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return "", fmt.Errorf("failed to open zip file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return "", fmt.Errorf("failed to extract file: %w", err)
		}
	}

	return extractDir, nil
}

// verifyChecksum 验证文件校验和
func (l *PluginLoader) verifyChecksum(filePath, expectedChecksum string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected=%s, actual=%s", expectedChecksum, actualChecksum)
	}

	return nil
}

// DiscoverPlugins 发现插件目录中的所有插件
func (l *PluginLoader) DiscoverPlugins() ([]string, error) {
	if _, err := os.Stat(l.pluginDir); os.IsNotExist(err) {
		return nil, nil
	}

	var plugins []string
	err := filepath.Walk(l.pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".xpkg") {
			plugins = append(plugins, path)
		}

		return nil
	})

	return plugins, err
}

