package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

type RancherUser struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

type UserCapabilities struct {
	CreateEnvironments bool `json:"createEnvironments"`
	ListRequests       bool `json:"listRequests"`
	ListAllRequests    bool `json:"listAllRequests"`
	Admin              bool `json:"admin"`
}

type AuthMeResponse struct {
	User         RancherUser      `json:"user"`
	AuthMode     string           `json:"authMode"`
	Capabilities UserCapabilities `json:"capabilities"`
	Error        string           `json:"error,omitempty"`
}

type requestUser struct {
	Token      string
	AuthMode   string
	User       RancherUser
	Kubeconfig string
}

const ctxRequestUser = "devportalRequestUser"

func tokenFromRequest(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if t := c.GetHeader("X-Rancher-Token"); t != "" {
		return t
	}
	if t := c.Query("token"); t != "" {
		return t
	}
	return ""
}

func allowServiceTokenFallback() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ALLOW_SERVICE_TOKEN")))
	return v == "" || v == "true" || v == "1" || v == "yes"
}

func resolveRequestToken(c *gin.Context) (string, string, error) {
	if tok := tokenFromRequest(c); tok != "" {
		return tok, "session", nil
	}
	if allowServiceTokenFallback() {
		if tok := rancherToken(); tok != "" {
			return tok, "service", nil
		}
	}
	return "", "", fmt.Errorf("authentication required: log into Rancher or pass Authorization: Bearer <token>")
}

func loadRequestUser(c *gin.Context) (*requestUser, error) {
	if v, ok := c.Get(ctxRequestUser); ok {
		if ru, ok := v.(*requestUser); ok {
			return ru, nil
		}
	}
	token, authMode, err := resolveRequestToken(c)
	if err != nil {
		return nil, err
	}
	user, err := fetchRancherUser(token)
	if err != nil {
		return nil, fmt.Errorf("invalid Rancher token: %w", err)
	}
	ru := &requestUser{
		Token:      token,
		AuthMode:   authMode,
		User:       user,
		Kubeconfig: kubeconfigPathForUser(user.ID),
	}
	c.Set(ctxRequestUser, ru)
	return ru, nil
}

func requireAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := loadRequestUser(c); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}
		c.Next()
	}
}

func isRancherAdmin(token, userID, authMode string) bool {
	if authMode == "service" {
		return true
	}
	user, err := fetchRancherUser(token)
	if err == nil {
		if strings.EqualFold(user.Username, "admin") {
			return true
		}
	}
	q := url.Values{}
	q.Set("userId", userID)
	body, err := rancherRequestWithToken("GET", "/v3/globalRoleBindings?"+q.Encode(), token)
	if err != nil {
		return false
	}
	var result struct {
		Data []struct {
			GlobalRoleID string `json:"globalRoleId"`
			RoleTemplateID string `json:"roleTemplateId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	for _, b := range result.Data {
		for _, id := range []string{b.GlobalRoleID, b.RoleTemplateID} {
			id = strings.ToLower(strings.TrimSpace(id))
			if id == "admin" || strings.Contains(id, "admin") {
				return true
			}
		}
	}
	return false
}

func evaluatePortalCapabilities(token, userID, authMode string) UserCapabilities {
	admin := isRancherAdmin(token, userID, authMode)
	return UserCapabilities{
		CreateEnvironments: true,
		ListRequests:       true,
		ListAllRequests:    admin,
		Admin:              admin,
	}
}

func handleAuthMe(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, AuthMeResponse{
		User:         ru.User,
		AuthMode:     ru.AuthMode,
		Capabilities: evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode),
	})
}

func requestUserFromContext(c *gin.Context) (*requestUser, bool) {
	v, ok := c.Get(ctxRequestUser)
	if !ok {
		return nil, false
	}
	ru, ok := v.(*requestUser)
	return ru, ok
}

func writeJSON(c *gin.Context, code int, v any) {
	c.Header("Content-Type", "application/json")
	c.Status(code)
	_ = json.NewEncoder(c.Writer).Encode(v)
}
