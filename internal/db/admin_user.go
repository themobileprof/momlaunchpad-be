package db

import (
	"context"
	"fmt"
)

// SetUserAdmin toggles the is_admin flag for a user.
func (db *DB) SetUserAdmin(ctx context.Context, userID string, isAdmin bool) error {
	result, err := db.ExecContext(ctx,
		`UPDATE users SET is_admin = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		isAdmin, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to set admin flag: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateUserPasswordHash sets the password hash for email/password login.
func (db *DB) UpdateUserPasswordHash(ctx context.Context, userID, passwordHash string) error {
	result, err := db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		passwordHash, userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
