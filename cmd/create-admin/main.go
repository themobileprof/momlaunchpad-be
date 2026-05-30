package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/themobileprof/momlaunchpad-be/internal/db"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const minPasswordLen = 8

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set (check your .env file)")
	}

	fmt.Println("MomLaunchpad — Admin setup")
	fmt.Println("========================")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	emailDefault := os.Getenv("ADMIN_EMAIL")
	email, err := promptString(reader, "Admin email", emailDefault)
	if err != nil {
		return err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return fmt.Errorf("email is required")
	}

	database, err := db.NewFromURL(databaseURL)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	existing, err := database.GetUserByEmail(ctx, email)
	if err != nil && err != db.ErrNotFound {
		return fmt.Errorf("lookup failed: %w", err)
	}

	if existing != nil {
		return promoteExisting(ctx, reader, database, existing)
	}

	return createNewAdmin(ctx, reader, database, email)
}

func promoteExisting(ctx context.Context, reader *bufio.Reader, database *db.DB, user *db.User) error {
	fmt.Printf("\nFound existing user: %s\n", user.Email)
	fmt.Printf("User ID: %s\n", user.ID)

	if user.IsAdmin {
		fmt.Println("Status: already an admin")
	} else {
		ok, err := promptYesNo(reader, "Promote this user to admin?", true)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Cancelled.")
			return nil
		}
		if err := database.SetUserAdmin(ctx, user.ID, true); err != nil {
			return fmt.Errorf("promote failed: %w", err)
		}
		fmt.Println("✓ User promoted to admin.")
	}

	hasPassword := user.PasswordHash != ""
	if !hasPassword {
		fmt.Println("This account has no password (likely OAuth-only).")
		ok, err := promptYesNo(reader, "Set a password for admin dashboard login?", true)
		if err != nil {
			return err
		}
		if ok {
			if err := setPassword(ctx, reader, database, user.ID); err != nil {
				return err
			}
		}
	} else {
		ok, err := promptYesNo(reader, "Reset password?", false)
		if err != nil {
			return err
		}
		if ok {
			if err := setPassword(ctx, reader, database, user.ID); err != nil {
				return err
			}
		}
	}

	printSuccess(user.Email, user.ID)
	return nil
}

func createNewAdmin(ctx context.Context, reader *bufio.Reader, database *db.DB, email string) error {
	fmt.Println("\nNo user with that email — creating a new admin account.")

	name, err := promptString(reader, "Display name", "Admin")
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Admin"
	}

	passwordDefault := os.Getenv("ADMIN_INITIAL_PASSWORD")
	password, err := promptPassword("Password (min 8 characters)", passwordDefault)
	if err != nil {
		return err
	}

	confirm, err := promptPassword("Confirm password", "")
	if err != nil {
		return err
	}
	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	displayName := name
	user := &db.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         &displayName,
		Language:     "en",
		IsAdmin:      true,
	}

	if err := database.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	fmt.Println("✓ Admin account created.")
	printSuccess(user.Email, user.ID)
	return nil
}

func setPassword(ctx context.Context, reader *bufio.Reader, database *db.DB, userID string) error {
	_ = reader
	password, err := promptPassword("New password (min 8 characters)", "")
	if err != nil {
		return err
	}
	confirm, err := promptPassword("Confirm new password", "")
	if err != nil {
		return err
	}
	if password != confirm {
		return fmt.Errorf("passwords do not match")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := database.UpdateUserPasswordHash(ctx, userID, string(hash)); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	fmt.Println("✓ Password updated.")
	return nil
}

func printSuccess(email, userID string) {
	fmt.Println()
	fmt.Println("Done. You can sign in with:")
	fmt.Printf("  • Admin dashboard: http://localhost:5174\n")
	fmt.Printf("  • API login:       POST /api/auth/login\n")
	fmt.Printf("  Email:   %s\n", email)
	fmt.Printf("  User ID: %s\n", userID)
}

func promptString(reader *bufio.Reader, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", label, defaultValue)
	} else {
		fmt.Printf("%s: ", label)
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

func promptYesNo(reader *bufio.Reader, question string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}
	fmt.Printf("%s %s: ", question, hint)

	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes, nil
	}
	return line == "y" || line == "yes", nil
}

func promptPassword(label, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Printf("%s (Enter to use ADMIN_INITIAL_PASSWORD from .env): ", label)
	} else {
		fmt.Printf("%s: ", label)
	}

	var password string
	if term.IsTerminal(int(syscall.Stdin)) {
		bytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", err
		}
		password = string(bytes)
	} else {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			return "", err
		}
		password = strings.TrimSpace(line)
	}

	if password == "" && defaultValue != "" {
		password = defaultValue
	}

	if len(password) < minPasswordLen {
		return "", fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	return password, nil
}
