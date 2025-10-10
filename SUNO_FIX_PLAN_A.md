# Suno API修复方案A - data直接返回数组

## 问题分析
前端调用 `response.data.map()` 报错，说明前端期望 `data` 字段直接是数组。

## 修改位置
`relay/relay_task.go:247-266` sunoFetchByIDRespBodyBuilder函数

## 修改内容
```go
func sunoFetchByIDRespBodyBuilder(c *gin.Context) (respBody []byte, taskResp *dto.TaskError) {
	taskId := c.Param("id")
	userId := c.GetInt("id")

	originTask, exist, err := model.GetByTaskId(userId, taskId)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "get_task_failed", http.StatusInternalServerError)
		return
	}
	if !exist {
		taskResp = service.TaskErrorWrapperLocal(errors.New("task_not_exist"), "task_not_exist", http.StatusBadRequest)
		return
	}

	// 直接返回Data字段（歌曲数组），而不是整个TaskDto对象
	var songsData json.RawMessage
	if originTask.Data != nil && len(originTask.Data) > 0 {
		songsData = originTask.Data
	} else {
		songsData = json.RawMessage("[]") // 空数组
	}

	respBody, err = json.Marshal(dto.TaskResponse[json.RawMessage]{
		Code: "success",
		Data: songsData,
	})
	return
}
```

## 返回格式
```json
{
  "code": "success",
  "data": [
    {
      "id": "song_id",
      "audio_url": "...",
      ...
    }
  ]
}
```
