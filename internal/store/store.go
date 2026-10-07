package store

import (
	"context"
	"errors"

	"github.com/Radiqeee/taskapi/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

// Store is implemented by Memory (local dev, tests) and Postgres (production).
// Every task method is scoped by userID so users never see each other's data.
type Store interface {
	CreateUser(ctx context.Context, email, passwordHash string) (model.User, error)
	GetUserByEmail(ctx context.Context, email string) (model.User, error)

	CreateTask(ctx context.Context, userID int64, title, description string) (model.Task, error)
	ListTasks(ctx context.Context, userID int64, f model.TaskFilter) ([]model.Task, error)
	GetTask(ctx context.Context, userID, id int64) (model.Task, error)
	UpdateTask(ctx context.Context, userID, id int64, u model.TaskUpdate) (model.Task, error)
	DeleteTask(ctx context.Context, userID, id int64) error
}
