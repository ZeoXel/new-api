# 重新编译与运行日志

```bash
# 在仓库根目录重新构建
go build -o new-api main.go

# 查看后台服务日志（示例：本地 server.log）
tail -f server.log
```

> 说明：项目包含多个调试工具（`debug_*.go`），直接 `go build ./...` 会因为重复的 `main` 函数报错，推荐按上述方式单独构建 `main.go`。需要调试工具时请进入 `debug_tools` 目录按需 `go run`。
