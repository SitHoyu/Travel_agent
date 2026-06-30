package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	const query = `
INSERT INTO users (
  username,
  password_hash,
  nickname,
  status
) VALUES (?, ?, ?, ?)
`

	result, err := r.db.ExecContext(ctx, query, user.Username, user.PasswordHash, nullIfEmpty(user.Nickname), user.Status)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.User{}, fmt.Errorf("username already exists")
		}
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return domain.User{}, fmt.Errorf("fetch inserted user id: %w", err)
	}

	user.ID = id
	return r.GetByID(ctx, id)
}

func (r *MySQLRepository) GetByID(ctx context.Context, id int64) (domain.User, error) {
	const query = `
SELECT id, username, password_hash, nickname, status, created_at, updated_at
FROM users
WHERE id = ?
`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanUser(row)
}

func (r *MySQLRepository) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	const query = `
SELECT id, username, password_hash, nickname, status, created_at, updated_at
FROM users
WHERE username = ?
`

	row := r.db.QueryRowContext(ctx, query, username)
	return scanUser(row)
}

type userScanner interface {
	Scan(dest ...any) error
}

func scanUser(scanner userScanner) (domain.User, error) {
	var user domain.User
	var nickname sql.NullString
	err := scanner.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&nickname,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, ErrUserNotFound
		}
		return domain.User{}, err
	}
	user.Nickname = nickname.String
	return user, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
