package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/Radiqeee/taskapi/internal/model"
)

type createTaskRequest struct {
	Title       string `json:"title" binding:"required,max=200"`
	Description string `json:"description" binding:"max=2000"`
}

type updateTaskRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=1,max=200"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	Done        *bool   `json:"done"`
}

type listTasksQuery struct {
	Done   *bool `form:"done"`
	Limit  int   `form:"limit,default=20" binding:"min=1,max=100"`
	Offset int   `form:"offset" binding:"min=0"`
}

func (h *Handler) listTasks(c *gin.Context) {
	var q listTasksQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		writeError(c, http.StatusBadRequest, "invalid query: "+err.Error())
		return
	}

	tasks, err := h.store.ListTasks(c.Request.Context(), c.GetInt64(ctxUserID), model.TaskFilter{
		Done: q.Done, Limit: q.Limit, Offset: q.Offset,
	})
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": tasks, "limit": q.Limit, "offset": q.Offset})
}

func (h *Handler) createTask(c *gin.Context) {
	var req createTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	task, err := h.store.CreateTask(c.Request.Context(), c.GetInt64(ctxUserID), req.Title, req.Description)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (h *Handler) getTask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	task, err := h.store.GetTask(c.Request.Context(), c.GetInt64(ctxUserID), id)
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) updateTask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	var req updateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.Title == nil && req.Description == nil && req.Done == nil {
		writeError(c, http.StatusBadRequest, "nothing to update")
		return
	}

	task, err := h.store.UpdateTask(c.Request.Context(), c.GetInt64(ctxUserID), id, model.TaskUpdate{
		Title: req.Title, Description: req.Description, Done: req.Done,
	})
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *Handler) deleteTask(c *gin.Context) {
	id, ok := taskID(c)
	if !ok {
		return
	}
	if err := h.store.DeleteTask(c.Request.Context(), c.GetInt64(ctxUserID), id); err != nil {
		writeStoreError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func taskID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(c, http.StatusBadRequest, "invalid task id")
		return 0, false
	}
	return id, true
}
