package controller

import (
	"net/http"
	"one-api/common"
	"one-api/model"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// ImportTokenRequest 外部密钥导入请求
type ImportTokenRequest struct {
	Key            string `json:"key" binding:"required"`             // 密钥（48字符，不含sk-前缀）
	ExternalUserId string `json:"external_user_id" binding:"required"` // 外部用户ID（如Supabase UUID）
	Name           string `json:"name"`                                // 密钥名称
	UnlimitedQuota bool   `json:"unlimited_quota"`                     // 是否无限额度
	RemainQuota    int    `json:"remain_quota"`                        // 初始额度
	Group          string `json:"group"`                               // 分组
}

// ImportToken 导入外部生成的Token
// POST /api/token/import
func ImportToken(c *gin.Context) {
	var req ImportTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 清理密钥：移除可能的sk-前缀
	key := strings.TrimPrefix(req.Key, "sk-")

	// 验证密钥格式（48字符，仅含数字和字母）
	if len(key) != 48 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "密钥长度必须为48字符",
		})
		return
	}

	// 验证字符集
	validKeyPattern := regexp.MustCompile(`^[0-9a-zA-Z]+$`)
	if !validKeyPattern.MatchString(key) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "密钥只能包含数字和字母",
		})
		return
	}

	// 检查密钥是否已存在
	existingToken, _ := model.GetTokenByKey(key, true)
	if existingToken != nil && existingToken.Id > 0 {
		// 密钥已存在，返回现有token信息
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "密钥已存在",
			"data": gin.H{
				"token_id":         existingToken.Id,
				"key":              existingToken.Key,
				"external_user_id": existingToken.ExternalUserId,
				"existed":          true,
			},
		})
		return
	}

	// 查找或创建内部用户
	// 优先通过external_user_id查找已有关联的token来获取user_id
	var userId int
	existingTokenByExternal := model.GetTokenByExternalUserId(req.ExternalUserId)
	if existingTokenByExternal != nil {
		userId = existingTokenByExternal.UserId
	} else {
		// 使用管理员账号（user_id=1）作为默认关联
		// 实际场景中可能需要创建对应的用户或使用其他策略
		userId = 1
	}

	// 设置默认值
	name := req.Name
	if name == "" {
		name = "imported"
	}

	// 创建Token
	token := model.Token{
		UserId:         userId,
		Key:            key,
		Status:         common.TokenStatusEnabled,
		Name:           name,
		CreatedTime:    common.GetTimestamp(),
		AccessedTime:   common.GetTimestamp(),
		ExpiredTime:    -1, // 永不过期
		RemainQuota:    req.RemainQuota,
		UnlimitedQuota: req.UnlimitedQuota,
		ExternalUserId: req.ExternalUserId,
		Group:          req.Group,
	}

	err := token.Insert()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "创建Token失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "导入成功",
		"data": gin.H{
			"token_id":         token.Id,
			"key":              token.Key,
			"external_user_id": token.ExternalUserId,
			"existed":          false,
		},
	})
}

// GetTokenByExternalUser 通过外部用户ID获取Token列表
// GET /api/token/external/:external_user_id
func GetTokenByExternalUser(c *gin.Context) {
	externalUserId := c.Param("external_user_id")
	if externalUserId == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "缺少external_user_id参数",
		})
		return
	}

	tokens, err := model.GetTokensByExternalUserId(externalUserId)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tokens,
	})
}
