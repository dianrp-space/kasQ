package db

import (
	"context"
	"log"

	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

func Seed(ctx context.Context, repo *repository.Repository) error {
	n, err := repo.CountUsers(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	team := &models.Team{
		Name:           "Kas Batam",
		Slug:           "kas-batam",
		InitialBalance: 2000000,
	}
	if err := repo.CreateTeam(ctx, team); err != nil {
		log.Printf("seed team: %v", err)
	}

	admin := &models.User{
		Name:          "Admin KasQ",
		Email:         "admin@kasq.local",
		PasswordHash:  string(hash),
		Role:          models.RoleAdmin,
		EmailVerified: true,
	}
	if err := repo.CreateUser(ctx, admin); err != nil {
		return err
	}

	opsTeamID := team.ID
	ops := &models.User{
		TeamID:        &opsTeamID,
		Name:          "Ops Batam",
		Email:         "ops@kasq.local",
		PasswordHash:  string(hash),
		Role:          models.RoleOps,
		EmailVerified: true,
	}
	return repo.CreateUser(ctx, ops)
}
