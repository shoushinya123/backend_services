package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var (
		inputDir  = flag.String("input", ".", "插件源码目录")
		output    = flag.String("output", "", "输出xpkg文件路径（必需）")
		manifest  = flag.String("manifest", "manifest.json", "manifest.json路径")
		pluginBin = flag.String("plugin", "plugin.so", "plugin.so路径")
	)
	flag.Parse()

	if *output == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定输出文件路径 (-output)\n")
		os.Exit(1)
	}

	// 检查必需文件
	if _, err := os.Stat(*manifest); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "错误: manifest.json 不存在: %s\n", *manifest)
		os.Exit(1)
	}

	if _, err := os.Stat(*pluginBin); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "错误: plugin.so 不存在: %s\n", *pluginBin)
		fmt.Fprintf(os.Stderr, "提示: 请先编译插件: go build -buildmode=plugin -o plugin.so plugin.go\n")
		os.Exit(1)
	}

	// 创建ZIP文件
	if err := createXpkg(*inputDir, *output, *manifest, *pluginBin); err != nil {
		fmt.Fprintf(os.Stderr, "错误: 打包失败: %v\n", err)
		os.Exit(1)
	}

	// 计算校验和
	checksum, err := calculateChecksum(*output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "警告: 计算校验和失败: %v\n", err)
	} else {
		fmt.Printf("✅ 插件打包成功: %s\n", *output)
		fmt.Printf("📦 校验和 (SHA256): %s\n", checksum)
	}
}

func createXpkg(inputDir, outputPath, manifestPath, pluginBin string) error {
	// 创建输出文件
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer zipFile.Close()

	// 创建ZIP写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 要包含的文件列表
	files := []string{
		manifestPath,
		pluginBin,
	}

	// 可选文件
	optionalFiles := []string{
		"README.md",
		"LICENSE",
		"config.schema.json",
	}

	for _, file := range optionalFiles {
		if _, err := os.Stat(file); err == nil {
			files = append(files, file)
		}
	}

	// 添加assets目录（如果存在）
	assetsDir := "assets"
	if info, err := os.Stat(assetsDir); err == nil && info.IsDir() {
		err := filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk assets dir: %w", err)
		}
	}

	// 添加文件到ZIP
	for _, file := range files {
		if err := addFileToZip(zipWriter, file); err != nil {
			return fmt.Errorf("failed to add file %s: %w", file, err)
		}
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// 使用文件名（不包含路径）
	header.Name = filepath.Base(filePath)
	if strings.HasPrefix(filePath, "assets/") {
		header.Name = filePath // 保留assets目录结构
	}

	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}

func calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

