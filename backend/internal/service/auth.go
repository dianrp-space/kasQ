package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/kasq/backend/internal/email"
	"github.com/kasq/backend/internal/models"
	"github.com/kasq/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo    *repository.Repository
	mailer  *email.Sender
	appURL  string
}

func NewAuthService(repo *repository.Repository, mailer *email.Sender, appURL string) *AuthService {
	return &AuthService{repo: repo, mailer: mailer, appURL: appURL}
}

func (a *AuthService) Register(ctx context.Context, name, emailAddr, password string) error {
	exists, err := a.repo.EmailExists(ctx, emailAddr)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("email sudah terdaftar")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		Name:         name,
		Email:        emailAddr,
		PasswordHash: string(hash),
		Role:         models.RoleOps,
		EmailVerified: false,
	}
	if err := a.repo.CreateUser(ctx, user); err != nil {
		return err
	}

	token := uuid.New().String()
	expires := time.Now().Add(24 * time.Hour)
	if err := a.repo.SetVerificationToken(ctx, user.ID, token, expires); err != nil {
		return err
	}

	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", a.appURL, token)
	subject, html, plain := email.VerificationEmail(name, verifyURL)
	if err := a.mailer.Send(emailAddr, subject, html, plain); err != nil {
		return err
	}
	log.Printf("auth: verification link for %s: %s", emailAddr, verifyURL)
	return nil
}

func (a *AuthService) VerifyEmail(ctx context.Context, token string) error {
	user, err := a.repo.GetUserByVerificationToken(ctx, token)
	if err != nil {
		return fmt.Errorf("token tidak valid atau sudah kadaluarsa")
	}
	return a.repo.VerifyUserEmail(ctx, user.ID)
}

func (a *AuthService) ForgotPassword(ctx context.Context, emailAddr string) error {
	user, err := a.repo.GetUserByEmail(ctx, emailAddr)
	if err != nil {
		// Don't leak whether email exists
		return nil
	}

	token := uuid.New().String()
	expires := time.Now().Add(time.Hour)
	if err := a.repo.SetResetToken(ctx, user.ID, token, expires); err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", a.appURL, token)
	subject, html, plain := email.ResetPasswordEmail(user.Name, resetURL)
	return a.mailer.Send(user.Email, subject, html, plain)
}

func (a *AuthService) ResetPassword(ctx context.Context, token, password string) error {
	if len(password) < 6 {
		return fmt.Errorf("password minimal 6 karakter")
	}
	user, err := a.repo.GetUserByResetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("token tidak valid atau sudah kadaluarsa")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return a.repo.UpdatePassword(ctx, user.ID, string(hash))
}

func (a *AuthService) ResendVerification(ctx context.Context, emailAddr string) error {
	user, err := a.repo.GetUserByEmail(ctx, emailAddr)
	if err != nil {
		return fmt.Errorf("email tidak ditemukan")
	}
	if user.EmailVerified {
		return fmt.Errorf("email sudah diverifikasi")
	}
	token := uuid.New().String()
	expires := time.Now().Add(24 * time.Hour)
	if err := a.repo.SetVerificationToken(ctx, user.ID, token, expires); err != nil {
		return err
	}
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", a.appURL, token)
	subject, html, plain := email.VerificationEmail(user.Name, verifyURL)
	if err := a.mailer.Send(user.Email, subject, html, plain); err != nil {
		return err
	}
	log.Printf("auth: verification link for %s: %s", user.Email, verifyURL)
	return nil
}
