package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"linkmeqr/backend/internal/models"
)

var ErrNotFound = errors.New("record not found")

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = ?`, email)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, `SELECT * FROM users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, role, full_name, phone, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.Email, u.PasswordHash, u.Role, u.FullName, u.Phone, u.IsActive,
	)
	return err
}

func (r *UserRepository) Update(ctx context.Context, u *models.User) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET full_name = ?, phone = ?, is_active = ?
		WHERE id = ?`,
		u.FullName, u.Phone, u.IsActive, u.ID,
	)
	return err
}

func (r *UserRepository) List(ctx context.Context, role models.Role) ([]models.User, error) {
	users := []models.User{}
	err := r.db.SelectContext(ctx, &users, `SELECT * FROM users WHERE role = ? ORDER BY created_at DESC`, role)
	return users, err
}

func (r *UserRepository) SetActive(ctx context.Context, id string, active bool) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET is_active = ? WHERE id = ?`, active, id)
	return err
}
