# 插件打包工具

用于将插件打包为xpkg格式的工具。

## 安装

```bash
cd tools/plugin-pack
go build -o plugin-pack main.go
```

## 使用方法

```bash
# 基本用法
./plugin-pack -input ./my-plugin -output my-plugin.xpkg

# 指定manifest和plugin文件
./plugin-pack \
  -input ./my-plugin \
  -output my-plugin.xpkg \
  -manifest manifest.json \
  -plugin plugin.so
```

## 参数说明

- `-input`: 插件源码目录（默认：当前目录）
- `-output`: 输出xpkg文件路径（必需）
- `-manifest`: manifest.json路径（默认：manifest.json）
- `-plugin`: plugin.so路径（默认：plugin.so）

## 打包流程

1. 检查必需文件（manifest.json、plugin.so）
2. 创建ZIP文件
3. 添加必需文件：
   - manifest.json
   - plugin.so
4. 添加可选文件（如果存在）：
   - README.md
   - LICENSE
   - config.schema.json
   - assets/目录
5. 计算SHA256校验和

## 示例

```bash
# 1. 编译插件
cd examples/plugins/dashscope
go build -buildmode=plugin -o plugin.so plugin.go

# 2. 打包插件
cd ../../../tools/plugin-pack
./plugin-pack -input ../../examples/plugins/dashscope -output ../../internal/plugin_storage/dashscope.xpkg
```

