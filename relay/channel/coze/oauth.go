package coze

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"one-api/common"
	relaycommon "one-api/relay/common"
	"one-api/service"
	"os"
	"strings"
	"time"

	"github.com/bytedance/gopkg/cache/asynccache"
	"github.com/golang-jwt/jwt"
)

type CozeOAuthConfig struct {
	AppID      string `json:"app_id"`
	KeyID      string `json:"key_id"`
	PrivateKey string `json:"private_key"`
	Aud        string `json:"aud"`
}

type CozeTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

var cozeOAuthCache = asynccache.NewAsyncCache(asynccache.Options{
	RefreshDuration: time.Minute * 55,
	EnableExpire:    true,
	ExpireDuration:  time.Minute * 50,
	Fetcher: func(key string) (interface{}, error) {
		return nil, errors.New("not found")
	},
})

func expandEnvVar(value string) string {
	if strings.HasPrefix(value, "$") {
		envVarName := strings.TrimPrefix(value, "$")
		envValue := os.Getenv(envVarName)
		if envValue != "" {
			return envValue
		}
	}
	return value
}

func ParseCozeOAuthConfig(key string) (*CozeOAuthConfig, error) {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "{") {
		return nil, errors.New("not a valid OAuth config JSON")
	}

	var config CozeOAuthConfig
	err := json.Unmarshal([]byte(key), &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OAuth config: %w", err)
	}

	if config.AppID == "" || config.KeyID == "" || config.PrivateKey == "" {
		return nil, errors.New("OAuth config is incomplete: app_id, key_id, and private_key are required")
	}

	config.PrivateKey = expandEnvVar(config.PrivateKey)

	if config.Aud == "" {
		config.Aud = "api.coze.com"
	}

	return &config, nil
}

func IsCozeOAuthConfig(key string) bool {
	_, err := ParseCozeOAuthConfig(key)
	return err == nil
}

func GetCozeAccessToken(info *relaycommon.RelayInfo, oauthConfig *CozeOAuthConfig) (string, error) {
	var cacheKey string
	if info.ChannelIsMultiKey {
		cacheKey = fmt.Sprintf("coze-oauth-token-%d-%d", info.ChannelId, info.ChannelMultiKeyIndex)
	} else {
		cacheKey = fmt.Sprintf("coze-oauth-token-%d", info.ChannelId)
	}

	val, err := cozeOAuthCache.Get(cacheKey)
	if err == nil {
		return val.(string), nil
	}

	signedJWT, err := createCozeSignedJWT(oauthConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create signed JWT: %w", err)
	}

	newToken, err := exchangeJWTForCozeAccessToken(signedJWT, oauthConfig, info)
	if err != nil {
		return "", fmt.Errorf("failed to exchange JWT for access token: %w", err)
	}

	if err := cozeOAuthCache.SetDefault(cacheKey, newToken); err {
		return newToken, nil
	}

	return newToken, nil
}

func createCozeSignedJWT(config *CozeOAuthConfig) (string, error) {
	privateKeyPEM := config.PrivateKey

	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "-----BEGIN PRIVATE KEY-----", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "-----END PRIVATE KEY-----", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\r", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\n", "")
	privateKeyPEM = strings.ReplaceAll(privateKeyPEM, "\\n", "")

	block, _ := pem.Decode([]byte("-----BEGIN PRIVATE KEY-----\n" + privateKeyPEM + "\n-----END PRIVATE KEY-----"))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("not an RSA private key")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":          config.AppID,
		"aud":          config.Aud,
		"iat":          now.Unix(),
		"exp":          now.Add(time.Hour).Unix(),
		"jti":          common.GetRandomString(16),
		"session_name": "coze-workflow-client",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = config.KeyID

	signedToken, err := token.SignedString(rsaPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedToken, nil
}

func exchangeJWTForCozeAccessToken(signedJWT string, config *CozeOAuthConfig, info *relaycommon.RelayInfo) (string, error) {
	tokenURL := fmt.Sprintf("%s/api/permission/oauth2/token", info.ChannelBaseUrl)

	payload := map[string]interface{}{
		"grant_type":       "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"duration_seconds": 900,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+signedJWT)

	var client *http.Client
	if info.ChannelSetting.Proxy != "" {
		client, err = service.NewProxyHttpClient(info.ChannelSetting.Proxy)
		if err != nil {
			return "", fmt.Errorf("failed to create proxy client: %w", err)
		}
	} else {
		client = service.GetHttpClient()
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp CozeTokenResponse
	err = json.Unmarshal(body, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("access token is empty in response")
	}

	return tokenResp.AccessToken, nil
}