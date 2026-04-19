package doc

import "embed"

// 使用 go:embed 指令，将当前目录下所有文件嵌入到变量 Assets 中

//go:embed swagger/*
var SwaggerFiles embed.FS
