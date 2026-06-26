package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	},
	Timeout: 30 * time.Second,
}

func rancherURL() string {
	if u := os.Getenv("RANCHER_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "https://rancher:443"
}

func rancherToken() string {
	return os.Getenv("RANCHER_TOKEN")
}

func rancherRequestWithToken(method, path, token string) ([]byte, error) {
	tok := token
	if tok == "" {
		tok = rancherToken()
	}
	if tok == "" {
		return nil, fmt.Errorf("no Rancher token")
	}
	url := rancherURL() + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("rancher API %s returned %d: %s", path, resp.StatusCode, string(body))
	}
	return body, nil
}

func fetchRancherUser(token string) (RancherUser, error) {
	body, err := rancherRequestWithToken("GET", "/v3/users?me=true", token)
	if err != nil {
		return RancherUser{}, err
	}
	var result struct {
		Data []struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return RancherUser{}, err
	}
	if len(result.Data) == 0 {
		return RancherUser{}, fmt.Errorf("no user returned from Rancher")
	}
	u := result.Data[0]
	name := u.DisplayName
	if name == "" {
		name = u.Name
	}
	if name == "" {
		name = u.Username
	}
	return RancherUser{ID: u.ID, Username: u.Username, DisplayName: name}, nil
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Rancher-Token")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(corsMiddleware())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	api.Use(requireAuthMiddleware())
	{
		api.GET("/auth/me", handleAuthMe)
		api.GET("/portal/catalog", handlePortalCatalog)
		api.GET("/portal/stack", handlePortalStack)
		api.GET("/portal/requests", handlePortalListRequests)
		api.POST("/portal/requests", handlePortalCreateRequest)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	r.Run(":" + port)
}
