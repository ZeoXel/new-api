package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"one-api/common"
	"one-api/constant"

	"github.com/gin-gonic/gin"
)

func executeKlingConvertRequest(t *testing.T, routePath string, body map[string]interface{}) map[string]interface{} {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST(routePath, KlingRequestConvert(), func(c *gin.Context) {
		resp := map[string]interface{}{
			"action":             c.GetString("action"),
			"path":               c.Request.URL.Path,
			"original_path":      c.GetString("bltcy_original_path"),
			"billing_model_name": c.GetString("billing_model_name"),
		}

		if raw, ok := c.Get(common.KeyRequestBody); ok {
			if payload, ok := raw.([]byte); ok && len(payload) > 0 {
				var unified map[string]interface{}
				_ = json.Unmarshal(payload, &unified)
				resp["unified"] = unified
			}
		}
		c.JSON(http.StatusOK, resp)
	})

	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, routePath, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status=%d body=%s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return resp
}

func TestKlingRequestConvertSetsActionByPath(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		body           map[string]interface{}
		expectedAction string
	}{
		{
			name: "text2video path",
			path: "/kling/v1/videos/text2video",
			body: map[string]interface{}{
				"model":  "kling-v3",
				"prompt": "text",
			},
			expectedAction: constant.TaskActionTextGenerate,
		},
		{
			name: "image2video path",
			path: "/kling/v1/videos/image2video",
			body: map[string]interface{}{
				"model":  "kling-v3",
				"prompt": "image",
			},
			expectedAction: constant.TaskActionGenerate,
		},
		{
			name: "omni-video path",
			path: "/kling/v1/videos/omni-video",
			body: map[string]interface{}{
				"model":  "kling-v3-omni",
				"prompt": "omni",
			},
			expectedAction: constant.TaskActionOmniVideo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := executeKlingConvertRequest(t, tt.path, tt.body)
			if action, _ := resp["action"].(string); action != tt.expectedAction {
				t.Fatalf("action mismatch, got=%q want=%q", action, tt.expectedAction)
			}
			if path, _ := resp["path"].(string); path != "/v1/video/generations" {
				t.Fatalf("path mismatch, got=%q want=/v1/video/generations", path)
			}
		})
	}
}

func TestKlingRequestConvertDefaultsTextActionWithoutMedia(t *testing.T) {
	resp := executeKlingConvertRequest(t, "/kling/v1/videos/custom", map[string]interface{}{
		"model":  "kling-v3",
		"prompt": "fallback",
	})

	if action, _ := resp["action"].(string); action != constant.TaskActionTextGenerate {
		t.Fatalf("fallback action mismatch, got=%q want=%q", action, constant.TaskActionTextGenerate)
	}
}

func TestKlingRequestConvertOmniPathToUnifiedRelayPath(t *testing.T) {
	resp := executeKlingConvertRequest(t, "/kling/v1/videos/omni-video", map[string]interface{}{
		"model_name": "kling-v3-omni",
		"prompt":     "omni prompt",
		"image_list": []interface{}{
			map[string]interface{}{"image": "img-a", "type": "subject"},
		},
	})

	if path, _ := resp["path"].(string); path != "/v1/video/generations" {
		t.Fatalf("path mismatch, got=%q want=/v1/video/generations", path)
	}
	if originalPath, _ := resp["original_path"].(string); originalPath != "/kling/v1/videos/omni-video" {
		t.Fatalf("original path mismatch, got=%q", originalPath)
	}
	if model, _ := resp["billing_model_name"].(string); model != "kling-v3-omni" {
		t.Fatalf("billing_model_name mismatch, got=%q", model)
	}

	unified, ok := resp["unified"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing unified payload")
	}
	metadata, ok := unified["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing metadata in unified payload")
	}
	if _, ok := metadata["image_list"]; !ok {
		t.Fatalf("metadata missing image_list: %v", metadata)
	}
}
