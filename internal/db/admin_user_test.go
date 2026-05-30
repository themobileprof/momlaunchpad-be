package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSetUserAdmin(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectExec(`UPDATE users SET is_admin`).
		WithArgs(true, "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	database := &DB{DB: sqlDB}
	if err := database.SetUserAdmin(context.Background(), "user-1", true); err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSetUserAdmin_NotFound(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectExec(`UPDATE users SET is_admin`).
		WithArgs(true, "missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	database := &DB{DB: sqlDB}
	err = database.SetUserAdmin(context.Background(), "missing", true)
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestUpdateUserPasswordHash(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectExec(`UPDATE users SET password_hash`).
		WithArgs("hash", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	database := &DB{DB: sqlDB}
	if err := database.UpdateUserPasswordHash(context.Background(), "user-1", "hash"); err != nil {
		t.Fatal(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateUserPasswordHash_NotFound(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectExec(`UPDATE users SET password_hash`).
		WithArgs("hash", "missing").
		WillReturnResult(sqlmock.NewResult(0, 0))

	database := &DB{DB: sqlDB}
	err = database.UpdateUserPasswordHash(context.Background(), "missing", "hash")
	if err != ErrNotFound {
		t.Fatalf("got %v, want ErrNotFound", err)
	}
}

func TestSetUserAdmin_DBError(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	mock.ExpectExec(`UPDATE users SET is_admin`).
		WithArgs(true, "user-1").
		WillReturnError(sql.ErrConnDone)

	database := &DB{DB: sqlDB}
	if err := database.SetUserAdmin(context.Background(), "user-1", true); err == nil {
		t.Fatal("expected error")
	}
}
