# 数据看板quota_data记录修复总结

## 📋 问题描述

工作流按次计费未能正确体现在数据看板中,导致数据看板只显示部分token信息,而实际的按次计费数据未被记录到`quota_data`表。

## 🔍 问题分析

### 数据对比

| 指标 | logs表 | quota_data表 | 差异 |
|------|--------|--------------|------|
| 工作流记录总数 | 105条 | 20条 | **85条缺失** |
| 总quota | 265,766,280 | 56,317,943 | **209M缺失** |
| 最新记录时间 | 2025-10-21 16:32 | 2025-10-17 15:00 | **4天差距** |
| coze-workflow-async | 54条日志 | 12条记录(quota=0) | **全部未正确记录** |

### 根本原因

**异步工作流的日志记录函数(`recordAsyncConsumeLog`)没有调用`LogQuotaData`函数**

在 `relay/channel/coze/async.go:594` 处,虽然创建了日志记录到`logs`表,但缺少向`quota_data`表记录数据的逻辑。

对比正常的日志记录流程(`model/log.go:195-199`):
```go
if common.DataExportEnabled {
    gopool.Go(func() {
        LogQuotaData(userId, username, params.ModelName, params.Quota, common.GetTimestamp(), params.PromptTokens+params.CompletionTokens)
    })
}
```

## ✅ 修复方案

在 `relay/channel/coze/async.go:601-607` 添加了缺失的`LogQuotaData`调用:

```go
// 记录到数据看板 quota_data 表
if common.DataExportEnabled {
    gopool.Go(func() {
        model.LogQuotaData(info.UserId, username, info.OriginModelName, quota, task.FinishTime, usage.PromptTokens+usage.CompletionTokens)
        common.SysLog(fmt.Sprintf("[Async] Logged quota data for task %s: quota=%d, tokens=%d", task.TaskID, quota, usage.PromptTokens+usage.CompletionTokens))
    })
}
```

### 修复位置

- **文件**: `relay/channel/coze/async.go`
- **函数**: `recordAsyncConsumeLog`
- **行数**: 601-607 (新增)

## 🚀 部署步骤

### 1. 编译修复版本
```bash
go build -ldflags "-s -w" -o new-api-fixed
```

### 2. 停止旧服务
```bash
killall new-api
```

### 3. 备份并替换
```bash
mv new-api new-api-old.backup
mv new-api-fixed new-api
```

### 4. 启动新服务
```bash
nohup ./new-api > new-api.log 2>&1 &
```

### 5. 验证服务
```bash
curl http://localhost:3000/api/status
ps aux | grep new-api
```

## 🧪 测试验证

### 方法1: 实时监控quota_data变化
```bash
./monitor_quota_data.sh
```

### 方法2: 查看日志输出
```bash
tail -f new-api.log | grep "Logged quota data"
```

### 方法3: 数据库查询
```bash
# 查询最新记录
sqlite3 data/one-api.db "SELECT model_name, quota, token_used, count, datetime(created_at, 'unixepoch', 'localtime') as time FROM quota_data ORDER BY created_at DESC LIMIT 10;"

# 统计总记录数
sqlite3 data/one-api.db "SELECT COUNT(*) FROM quota_data;"
```

### 预期结果

修复后,异步工作流每次成功执行完成时,应该看到:

1. **日志输出**:
   ```
   [Async] Successfully created log for task xxx with xxx tokens
   [Async] Logged quota data for task xxx: quota=xxxxx, tokens=xxx
   ```

2. **quota_data表新增记录**:
   - model_name: 工作流ID或模型名称
   - quota: 按次计费的金额
   - token_used: 实际消耗的token数
   - count: 调用次数(1次)

3. **数据看板显示**:
   - "统计额度"卡片显示正确的quota总和
   - "统计Tokens"卡片显示正确的token总和
   - 图表中包含工作流的消耗数据

## 📊 影响范围

### 受影响的功能
- ✅ 异步工作流 (`coze-workflow-async`)
- ✅ 按次计费的工作流
- ✅ 数据看板统计

### 不受影响的功能
- ✅ 同步工作流(已正确记录)
- ✅ 普通对话模型
- ✅ 其他渠道适配器
- ✅ 余额扣除逻辑
- ✅ 使用日志显示

## 🔄 回滚方案

如果修复后出现问题,可以快速回滚:

```bash
killall new-api
mv new-api new-api-fixed.backup
mv new-api-old.backup new-api
nohup ./new-api > new-api.log 2>&1 &
```

## 📝 后续优化建议

1. **统一日志记录**: 将异步和同步的日志记录逻辑合并到统一函数
2. **监控告警**: 添加quota_data记录失败的告警机制
3. **数据补录**: 考虑为历史缺失的数据编写补录脚本
4. **单元测试**: 添加quota_data记录的单元测试

## 🎯 验证清单

- [x] 编译成功
- [x] 服务启动正常
- [x] 接口响应正常
- [ ] 发起异步工作流请求
- [ ] 确认日志中有"Logged quota data"输出
- [ ] 验证quota_data表有新记录
- [ ] 检查数据看板显示正确

## 📅 修复时间

- **问题发现**: 2025-10-23
- **修复完成**: 2025-10-23
- **部署时间**: 2025-10-23 11:52

---

**修复状态**: ✅ 已完成
**验证状态**: ⏳ 待验证(需要实际工作流请求触发)
