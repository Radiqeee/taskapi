package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Radiqeee/taskapi/internal/model"
)

var schema = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id            BIGSERIAL PRIMARY KEY,
		email         TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	`CREATE TABLE IF NOT EXISTS tasks (
		id          BIGSERIAL PRIMARY KEY,
		user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title       TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		done        BOOLEAN NOT NULL DEFAULT false,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
	)`,
	`CREATE INDEX IF NOT EXISTS tasks_user_id_idx ON tasks (user_id, id DESC)`,
}

const taskColumns = `id, user_id, title, description, done, created_at, updated_at`

type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres connects, verifies the connection and applies the schema.
func NewPostgres(ctx context.Context, url string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	for _, stmt := range schema {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			pool.Close()
			return nil, fmt.Errorf("migrate: %w", err)
		}
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() { p.pool.Close() }

func (p *Postgres) CreateUser(ctx context.Context, email, passwordHash string) (model.User, error) {
	var u model.User
	err := p.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return model.User{}, ErrConflict
	}
	return u, err
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	var u model.User
	err := p.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, mapErr(err)
}

func (p *Postgres) CreateTask(ctx context.Context, userID int64, title, description string) (model.Task, error) {
	row := p.pool.QueryRow(ctx,
		`INSERT INTO tasks (user_id, title, description) VALUES ($1, $2, $3) RETURNING `+taskColumns,
		userID, title, description,
	)
	return scanTask(row)
}

func (p *Postgres) ListTasks(ctx context.Context, userID int64, f model.TaskFilter) ([]model.Task, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT `+taskColumns+` FROM tasks
		 WHERE user_id = $1 AND ($2::boolean IS NULL OR done = $2)
		 ORDER BY id DESC LIMIT $3 OFFSET $4`,
		userID, f.Done, f.Limit, f.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []model.Task{}
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (p *Postgres) GetTask(ctx context.Context, userID, id int64) (model.Task, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE id = $1 AND user_id = $2`, id, userID,
	)
	return scanTask(row)
}

func (p *Postgres) UpdateTask(ctx context.Context, userID, id int64, u model.TaskUpdate) (model.Task, error) {
	row := p.pool.QueryRow(ctx,
		`UPDATE tasks SET
			title       = COALESCE($3::text, title),
			description = COALESCE($4::text, description),
			done        = COALESCE($5::boolean, done),
			updated_at  = now()
		 WHERE id = $1 AND user_id = $2
		 RETURNING `+taskColumns,
		id, userID, u.Title, u.Description, u.Done,
	)
	return scanTask(row)
}

func (p *Postgres) DeleteTask(ctx context.Context, userID, id int64) error {
	tag, err := p.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func scanTask(row pgx.Row) (model.Task, error) {
	var t model.Task
	err := row.Scan(&t.ID, &t.UserID, &t.Title, &t.Description, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	return t, mapErr(err)
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
