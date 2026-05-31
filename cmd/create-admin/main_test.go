package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
)

func testWithStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
	go func() {
		_, _ = io.WriteString(w, input)
		_ = w.Close()
	}()
	fn()
}

func newTestDB(t *testing.T) (*db.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return &db.DB{DB: sqlDB}, mock
}

func TestPromptString_UsesDefault(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	got, err := promptString(reader, "Email", "admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "admin@example.com" {
		t.Fatalf("got %q, want default", got)
	}
}

func TestPromptString_CustomValue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("custom@example.com\n"))
	got, err := promptString(reader, "Email", "admin@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if got != "custom@example.com" {
		t.Fatalf("got %q", got)
	}
}

func TestPromptYesNo_DefaultYes(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	ok, err := promptYesNo(reader, "Promote?", true)
	if err != nil || !ok {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
}

func TestPromptYesNo_ExplicitNo(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("n\n"))
	ok, err := promptYesNo(reader, "Promote?", true)
	if err != nil || ok {
		t.Fatalf("got ok=%v err=%v", ok, err)
	}
}

func TestPromptPassword_TooShort(t *testing.T) {
	testWithStdin(t, "short\n", func() {
		_, err := promptPassword("Password", "")
		if err == nil {
			t.Fatal("expected error for short password")
		}
	})
}

func TestPromptPassword_AcceptsMinLength(t *testing.T) {
	testWithStdin(t, "password123\n", func() {
		got, err := promptPassword("Password", "")
		if err != nil {
			t.Fatal(err)
		}
		if got != "password123" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestPromptPassword_UsesDefaultFromEnv(t *testing.T) {
	testWithStdin(t, "\n", func() {
		got, err := promptPassword("Password", "defaultpass")
		if err != nil {
			t.Fatal(err)
		}
		if got != "defaultpass" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestCreateNewAdmin_Success(t *testing.T) {
	database, mock := newTestDB(t)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("admin@example.com", sqlmock.AnyArg(), "Admin User", "en", true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("admin-id-1", now, now))

	reader := bufio.NewReader(strings.NewReader("Admin User\n"))
	testWithStdin(t, "password123\npassword123\n", func() {
		if err := createNewAdmin(ctx, reader, database, "admin@example.com"); err != nil {
			t.Fatal(err)
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateNewAdmin_PasswordMismatch(t *testing.T) {
	database, _ := newTestDB(t)
	reader := bufio.NewReader(strings.NewReader("Admin\n"))
	testWithStdin(t, "password123\notherpass99\n", func() {
		err := createNewAdmin(context.Background(), reader, database, "admin@example.com")
		if err == nil || !strings.Contains(err.Error(), "do not match") {
			t.Fatalf("got err = %v", err)
		}
	})
}

func TestPromoteExisting_AlreadyAdmin(t *testing.T) {
	database, mock := newTestDB(t)
	user := &db.User{ID: "user-1", Email: "admin@example.com", IsAdmin: true, PasswordHash: "hash"}

	// Decline password reset.
	reader := bufio.NewReader(strings.NewReader("n\n"))
	if err := promoteExisting(context.Background(), reader, database, user); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPromoteExisting_PromoteNonAdmin(t *testing.T) {
	database, mock := newTestDB(t)
	user := &db.User{ID: "user-1", Email: "user@example.com", IsAdmin: false, PasswordHash: "hash"}

	mock.ExpectExec(`UPDATE users SET is_admin`).
		WithArgs(true, "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	reader := bufio.NewReader(strings.NewReader("y\nn\n"))
	if err := promoteExisting(context.Background(), reader, database, user); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPromoteExisting_CancelPromotion(t *testing.T) {
	database, mock := newTestDB(t)
	user := &db.User{ID: "user-1", Email: "user@example.com", IsAdmin: false, PasswordHash: "hash"}

	reader := bufio.NewReader(strings.NewReader("n\n"))
	if err := promoteExisting(context.Background(), reader, database, user); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPromoteExisting_OAuthUserSetsPassword(t *testing.T) {
	database, mock := newTestDB(t)
	user := &db.User{ID: "user-1", Email: "oauth@example.com", IsAdmin: true, PasswordHash: ""}

	mock.ExpectExec(`UPDATE users SET password_hash`).
		WithArgs(sqlmock.AnyArg(), "user-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	reader := bufio.NewReader(strings.NewReader("y\n"))
	testWithStdin(t, "password123\npassword123\n", func() {
		if err := promoteExisting(context.Background(), reader, database, user); err != nil {
			t.Fatal(err)
		}
	})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
