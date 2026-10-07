package store

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/Radiqeee/taskapi/internal/model"
)

type Memory struct {
	mu       sync.RWMutex
	users    map[int64]model.User
	tasks    map[int64]model.Task
	nextUser int64
	nextTask int64
}

func NewMemory() *Memory {
	return &Memory{
		users: make(map[int64]model.User),
		tasks: make(map[int64]model.Task),
	}
}

func (m *Memory) CreateUser(_ context.Context, email, passwordHash string) (model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.users {
		if u.Email == email {
			return model.User{}, ErrConflict
		}
	}
	m.nextUser++
	u := model.User{ID: m.nextUser, Email: email, PasswordHash: passwordHash, CreatedAt: time.Now().UTC()}
	m.users[u.ID] = u
	return u, nil
}

func (m *Memory) GetUserByEmail(_ context.Context, email string) (model.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return model.User{}, ErrNotFound
}

func (m *Memory) CreateTask(_ context.Context, userID int64, title, description string) (model.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextTask++
	now := time.Now().UTC()
	t := model.Task{ID: m.nextTask, UserID: userID, Title: title, Description: description, CreatedAt: now, UpdatedAt: now}
	m.tasks[t.ID] = t
	return t, nil
}

func (m *Memory) ListTasks(_ context.Context, userID int64, f model.TaskFilter) ([]model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := []model.Task{}
	for _, t := range m.tasks {
		if t.UserID != userID || (f.Done != nil && t.Done != *f.Done) {
			continue
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })

	if f.Offset >= len(out) {
		return []model.Task{}, nil
	}
	out = out[f.Offset:]
	if f.Limit > 0 && f.Limit < len(out) {
		out = out[:f.Limit]
	}
	return out, nil
}

func (m *Memory) GetTask(_ context.Context, userID, id int64) (model.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.tasks[id]
	if !ok || t.UserID != userID {
		return model.Task{}, ErrNotFound
	}
	return t, nil
}

func (m *Memory) UpdateTask(_ context.Context, userID, id int64, u model.TaskUpdate) (model.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok || t.UserID != userID {
		return model.Task{}, ErrNotFound
	}
	if u.Title != nil {
		t.Title = *u.Title
	}
	if u.Description != nil {
		t.Description = *u.Description
	}
	if u.Done != nil {
		t.Done = *u.Done
	}
	t.UpdatedAt = time.Now().UTC()
	m.tasks[id] = t
	return t, nil
}

func (m *Memory) DeleteTask(_ context.Context, userID, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok || t.UserID != userID {
		return ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}
