package kling

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"one-api/constant"
	relaycommon "one-api/relay/common"
	"one-api/service"

	"github.com/gin-gonic/gin"
)

func newRelayInfoForAction(action, apiKey string) *relaycommon.RelayInfo {
	return &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey: apiKey,
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: action,
		},
	}
}

func TestBuildRequestURLWithOmniAction(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		apiKey   string
		expected string
	}{
		{
			name:     "image2video",
			action:   constant.TaskActionGenerate,
			apiKey:   "ak|sk",
			expected: "https://example.com/v1/videos/image2video",
		},
		{
			name:     "text2video",
			action:   constant.TaskActionTextGenerate,
			apiKey:   "ak|sk",
			expected: "https://example.com/v1/videos/text2video",
		},
		{
			name:     "omni-video",
			action:   constant.TaskActionOmniVideo,
			apiKey:   "ak|sk",
			expected: "https://example.com/v1/videos/omni-video",
		},
		{
			name:     "omni-video with new-api relay prefix",
			action:   constant.TaskActionOmniVideo,
			apiKey:   "sk-new-api",
			expected: "https://example.com/kling/v1/videos/omni-video",
		},
	}

	a := &TaskAdaptor{baseURL: "https://example.com"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := newRelayInfoForAction(tt.action, tt.apiKey)
			got, err := a.BuildRequestURL(info)
			if err != nil {
				t.Fatalf("BuildRequestURL returned error: %v", err)
			}
			if got != tt.expected {
				t.Fatalf("BuildRequestURL mismatch, got=%q want=%q", got, tt.expected)
			}
		})
	}
}

func TestFetchTaskUsesOmniVideoPath(t *testing.T) {
	service.InitHttpClient()

	tests := []struct {
		name             string
		key              string
		taskID           string
		expectedPath     string
		expectAuthPrefix string
	}{
		{
			name:             "legacy key",
			key:              "ak|sk",
			taskID:           "task-legacy",
			expectedPath:     "/v1/videos/omni-video/task-legacy",
			expectAuthPrefix: "Bearer ",
		},
		{
			name:             "new-api key",
			key:              "sk-new-api",
			taskID:           "task-new-api",
			expectedPath:     "/kling/v1/videos/omni-video/task-new-api",
			expectAuthPrefix: "Bearer sk-new-api",
		},
	}

	a := &TaskAdaptor{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath, gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotAuth = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0}`))
			}))
			defer srv.Close()

			resp, err := a.FetchTask(srv.URL, tt.key, map[string]any{
				"task_id": tt.taskID,
				"action":  constant.TaskActionOmniVideo,
			})
			if err != nil {
				t.Fatalf("FetchTask returned error: %v", err)
			}
			defer resp.Body.Close()
			_, _ = io.ReadAll(resp.Body)

			if gotPath != tt.expectedPath {
				t.Fatalf("FetchTask path mismatch, got=%q want=%q", gotPath, tt.expectedPath)
			}
			if !strings.HasPrefix(gotAuth, tt.expectAuthPrefix) {
				t.Fatalf("authorization mismatch, got=%q want-prefix=%q", gotAuth, tt.expectAuthPrefix)
			}
		})
	}
}

func TestBuildRequestBodyIncludesOmniFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	req := relaycommon.TaskSubmitReq{
		Prompt: "test",
		Model:  "kling-v3-omni",
		Metadata: map[string]interface{}{
			"image_list": []interface{}{
				map[string]interface{}{"image": "img-a", "type": "subject"},
			},
			"video_list": []interface{}{
				map[string]interface{}{"video": "https://example.com/v.mp4", "type": "base"},
			},
			"element_list": []interface{}{
				map[string]interface{}{"image": "img-element"},
			},
			"multi_shot": true,
			"shot_type":  "customize",
			"multi_prompt": []interface{}{
				map[string]interface{}{"prompt": "shot-1", "duration": 5},
			},
			"sound": "on",
			"voice_list": []interface{}{
				map[string]interface{}{"id": "voice-1"},
			},
		},
	}
	c.Set("task_request", req)

	a := &TaskAdaptor{}
	bodyReader, err := a.BuildRequestBody(c, &relaycommon.RelayInfo{})
	if err != nil {
		t.Fatalf("BuildRequestBody returned error: %v", err)
	}

	payloadBytes, err := io.ReadAll(bodyReader)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	expectedKeys := []string{
		"image_list",
		"video_list",
		"element_list",
		"multi_shot",
		"shot_type",
		"multi_prompt",
		"sound",
		"voice_list",
	}
	for _, k := range expectedKeys {
		if _, ok := payload[k]; !ok {
			t.Fatalf("payload missing key %q, payload=%v", k, payload)
		}
	}

	if action := c.GetString("action"); action != constant.TaskActionOmniVideo {
		t.Fatalf("expected action to be %s, got %q", constant.TaskActionOmniVideo, action)
	}
}

func TestGetModelListReturnsLatestKlingModels(t *testing.T) {
	a := &TaskAdaptor{}
	got := a.GetModelList()
	expected := []string{"kling-v3", "kling-v2-6", "kling-v3-omni", "kling-video-o1"}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("GetModelList mismatch, got=%v want=%v", got, expected)
	}
}
