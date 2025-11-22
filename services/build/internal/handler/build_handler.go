package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/collab/services/build/internal/service"
)

func CreateBuild(buildService *service.BuildService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			ProjectID   string `json:"project_id" binding:"required"`
			CommitSHA   string `json:"commit_sha" binding:"required"`
			Branch      string `json:"branch" binding:"required"`
			TriggeredBy string `json:"triggered_by"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		build, err := buildService.CreateBuild(c.Request.Context(), req.ProjectID, req.CommitSHA, req.Branch, req.TriggeredBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, build)
	}
}

func GetBuild(buildService *service.BuildService) gin.HandlerFunc {
	return func(c *gin.Context) {
		buildID := c.Param("id")

		build, err := buildService.GetBuild(c.Request.Context(), buildID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Build not found"})
			return
		}

		c.JSON(http.StatusOK, build)
	}
}

func ListBuilds(buildService *service.BuildService) gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.Query("project_id")

		builds, err := buildService.ListBuilds(c.Request.Context(), projectID, 50, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"builds": builds})
	}
}

func CancelBuild(buildService *service.BuildService) gin.HandlerFunc {
	return func(c *gin.Context) {
		buildID := c.Param("id")

		if err := buildService.CancelBuild(c.Request.Context(), buildID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

func GetBuildLogs(buildService *service.BuildService) gin.HandlerFunc {
	return func(c *gin.Context) {
		buildID := c.Param("id")

		logs, err := buildService.GetBuildLogs(c.Request.Context(), buildID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"logs": logs})
	}
}

func GetBuildArtifacts(buildService *service.BuildService) gin.HandlerFunc {
	return func(c *gin.Context) {
		buildID := c.Param("id")

		artifacts, err := buildService.GetBuildArtifacts(c.Request.Context(), buildID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"artifacts": artifacts})
	}
}
