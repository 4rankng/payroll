package seed

import (
	"context"
	"fmt"

	"api-server/internal/config"
	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/password"

	"gorm.io/gorm"
)

// EnsureAdminUser creates or updates the default admin user.
func EnsureAdminUser(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	userRepo := persistence.NewUserRepository(&persistence.Database{DB: db})

	const (
		username     = "admin"
		email        = "admin@example.com"
		userPassword = "Vfic1234@"
		fullname     = "System Administrator"
	)

	user, err := userRepo.GetByUsername(ctx, username)
	if err != nil {
		if de, ok := err.(*domain.DomainError); ok && de.Type == "NOT_FOUND" {
			// Create new admin user
			hashed, hErr := password.HashPassword(userPassword, cfg.Security.HashSecret, cfg.Security.HashSalt)
			if hErr != nil {
				return fmt.Errorf("hash error: %w", hErr)
			}
			emailPtr := email
			u := &domain.User{Username: username, Email: &emailPtr, Password: hashed, Fullname: fullname, Role: domain.RoleAdmin}
			if cErr := userRepo.Create(ctx, u); cErr != nil {
				return fmt.Errorf("create admin error: %w", cErr)
			}
			fmt.Println("admin user created")
			return nil
		}
		return fmt.Errorf("lookup error: %w", err)
	}

	// Update existing user fields
	if user.Role != domain.RoleAdmin {
		user.Role = domain.RoleAdmin
	}
	if user.Fullname != fullname {
		user.Fullname = fullname
	}
	if user.Email == nil || *user.Email != email {
		emailPtr := email
		user.Email = &emailPtr
	}

	// Re-hash / set password unconditionally to guarantee desired password
	hashed, hErr := password.HashPassword(userPassword, cfg.Security.HashSecret, cfg.Security.HashSalt)
	if hErr != nil {
		return fmt.Errorf("hash error: %w", hErr)
	}
	user.Password = hashed

	// Always update since we re-hash password unconditionally
	if uErr := userRepo.Update(ctx, user); uErr != nil {
		return fmt.Errorf("update admin error: %w", uErr)
	}
	fmt.Println("admin user upserted (updated)")
	return nil
}

// EnsureAccounts creates or updates the standard chart of accounts.
func EnsureAccounts(ctx context.Context, db *gorm.DB) error {
	seeder := NewSeeder(db)
	return seeder.seedAccounts(ctx, db)
}
