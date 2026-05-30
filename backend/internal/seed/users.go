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

func (s *Seeder) seedPartnerUsers(ctx context.Context, db *gorm.DB) error {
	return s.seedUsersInBatches(ctx, db, 5, domain.RolePartner, "user")
}

// seedUsersInBatches generates users in memory-efficient batches
func (s *Seeder) seedUsersInBatches(ctx context.Context, db *gorm.DB, totalUsers int, role domain.UserRole, usernamePrefix string) error {
	userRepo := persistence.NewUserRepository(&persistence.Database{DB: db})

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config for password hashing: %w", err)
	}

	batchSize := min(50, totalUsers) // Process in batches of 50 or less
	for offset := 0; offset < totalUsers; offset += batchSize {
		remaining := min(batchSize, totalUsers-offset)
		if err := s.generateUserBatch(ctx, db, userRepo, cfg, remaining, role, usernamePrefix); err != nil {
			return fmt.Errorf("batch %d-%d: %w", offset, offset+remaining, err)
		}
	}
	return nil
}

// generateUserBatch creates a single batch of users with collision avoidance
func (s *Seeder) generateUserBatch(ctx context.Context, db *gorm.DB, userRepo domain.UserRepository, cfg *config.Config, count int, role domain.UserRole, usernamePrefix string) error {
	// Initialize collision tracking maps for this batch
	existingUsernames := make(map[string]bool)
	existingEmails := make(map[string]bool)
	existingCCCDs := make(map[string]bool)

	// Load existing data to avoid collisions
	var existingUsers []domain.User
	if err := db.Select("username, email").Find(&existingUsers).Error; err != nil {
		return fmt.Errorf("check existing users: %w", err)
	}

	for _, user := range existingUsers {
		existingUsernames[user.Username] = true
		if user.Email != nil {
			existingEmails[*user.Email] = true
		}
	}

	// Generate and create users for this batch
	for i := 0; i < count; i++ {
		// Generate realistic user profile
		profile := GenerateRealisticUserProfile(existingUsernames, existingEmails, existingCCCDs)

		// Override username with prefix if provided
		if usernamePrefix != "" {
			profile.Username = GenerateUniqueUsernameWithDB(ctx, db, usernamePrefix, existingUsernames)
			profile.Email = GenerateUniqueEmailWithDB(ctx, db, profile.FullName, existingEmails)
		}

		// Always create user with unique profile (no duplicate checking)
		hashedPw, hErr := password.HashPassword("Vfic1234@", cfg.Security.HashSecret, cfg.Security.HashSalt)
		if hErr != nil {
			return fmt.Errorf("hash password: %w", hErr)
		}

		user := &domain.User{
			Username: profile.Username,
			Email:    &profile.Email,
			Password: hashedPw,
			Fullname: profile.FullName,
			Role:     role,
		}

		if err := userRepo.Create(ctx, user); err != nil {
			return fmt.Errorf("create user %s: %w", profile.Username, err)
		}
	}

	// Clear maps to free memory after batch processing
	clear(existingUsernames)
	clear(existingEmails)
	clear(existingCCCDs)

	return nil
}
