package swagger

import "embed"

//go:embed *.json *.html swagger-ui/*
var Files embed.FS
