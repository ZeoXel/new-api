# Suno API修复方案B - data包含任务信息，但songs字段是数组

## 问题分析
前端期望返回任务对象，但其中有个数组字段（如songs）供map使用。

## 修改dto.TaskDto结构
在 `dto/suno.go` 添加新的查询响应结构：

```go
type SunoTaskQueryResponse struct {
	TaskID     string          `json:"task_id"`
	Action     string          `json:"action"`
	Status     string          `json:"status"`
	FailReason string          `json:"fail_reason"`
	SubmitTime int64           `json:"submit_time"`
	StartTime  int64           `json:"start_time"`
	FinishTime int64           `json:"finish_time"`
	Progress   string          `json:"progress"`
	Songs      []SunoSong      `json:"songs"` // 解析后的歌曲数组
}
```

## 修改查询接口
`relay/relay_task.go:247-266`:

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

	// 解析Data字段为歌曲数组
	var songs []dto.SunoSong
	if originTask.Data != nil && len(originTask.Data) > 0 {
		err = json.Unmarshal(originTask.Data, &songs)
		if err != nil {
			// 解析失败，返回空数组
			songs = []dto.SunoSong{}
		}
	} else {
		songs = []dto.SunoSong{}
	}

	queryResp := dto.SunoTaskQueryResponse{
		TaskID:     originTask.TaskID,
		Action:     originTask.Action,
		Status:     string(originTask.Status),
		FailReason: originTask.FailReason,
		SubmitTime: originTask.SubmitTime,
		StartTime:  originTask.StartTime,
		FinishTime: originTask.FinishTime,
		Progress:   originTask.Progress,
		Songs:      songs,
	}

	respBody, err = json.Marshal(dto.TaskResponse[dto.SunoTaskQueryResponse]{
		Code: "success",
		Data: queryResp,
	})
	return
}
```

## 返回格式
```json
{
  "code": "success",
  "data": {
    "task_id": "xxx",
    "status": "SUCCESS",
    "songs": [
      {
        "id": "song_id",
        "audio_url": "...",
        ...
      }
    ]
  }
}
```
