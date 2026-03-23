package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dariamoshkina/gopherMart/internal/model"
	"github.com/dariamoshkina/gopherMart/internal/service"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, login, passwordHash string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (login, password_hash) VALUES ($1, $2)
		 RETURNING id, login, password_hash, created_at`,
		login, passwordHash,
	)
	var user model.User
	err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, service.ErrLoginTaken
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, login, password_hash, created_at FROM users WHERE login = $1`,
		login,
	)
	var user model.User
	if err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, service.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
