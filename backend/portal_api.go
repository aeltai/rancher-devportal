package main

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func handlePortalClusters(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	clusters, err := fetchClustersWithToken(ru.Token)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"clusters": clusters})
}

func handlePortalExistingResources(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if err := ensureClusterReady(ru); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	clusterID := strings.TrimSpace(c.Query("cluster"))
	if clusterID == "" {
		clusterID = "local"
	}
	items, err := listExistingNamespaces(ru.Kubeconfig, clusterID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"resources": items, "clusterId": clusterID})
}

type testGitConnectionBody struct {
	URL        string `json:"url"`
	Branch     string `json:"branch"`
	SecretName string `json:"secretName"`
}

func handlePortalTestGitConnection(c *gin.Context) {
	ru, err := loadRequestUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if !evaluatePortalCapabilities(ru.Token, ru.User.ID, ru.AuthMode).Admin {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin required"})
		return
	}
	if err := ensureClusterReady(ru); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	var body testGitConnectionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	branch := body.Branch
	if branch == "" {
		branch = "main"
	}
	if err := testGitConnection(body.URL, branch, body.SecretName, ru.Kubeconfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Git connection successful"})
}

func decodeSecretData(data map[string]string) map[string]string {
	out := make(map[string]string, len(data))
	for k, v := range data {
		if b, err := base64.StdEncoding.DecodeString(v); err == nil {
			out[k] = string(b)
		} else {
			out[k] = v
		}
	}
	return out
}
