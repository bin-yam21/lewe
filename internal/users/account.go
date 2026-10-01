package users

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/yeabt/lewe/internal/auth"
	"github.com/yeabt/lewe/internal/db"
	"github.com/yeabt/lewe/internal/mail"
	"github.com/yeabt/lewe/internal/validator"
)

const (
	verifyEmailTTL   = 48 * time.Hour
	resetPasswordTTL = time.Hour
)

var (
	ErrInvalidAccountToken = errors.New("invalid or expired link")
	ErrAlreadyVerified     = errors.New("email already verified")
)

// validatePassword applies the password rules used everywhere a password is set.
func validatePassword(errs validator.Errors, field, password string) {
	validator.ValidateRequired(errs, field, password)
	validator.ValidateMinLength(errs, field, password, 8)
	// bcrypt ignores everything past 72 bytes (and rejects longer input).
	validator.ValidateMaxLength(errs, field, password, 72)
}

// issueAccountToken creates and stores a one-time token, returning the raw
// value to put in an emailed link.
func (s *Service) issueAccountToken(ctx context.Context, userID pgtype.UUID, purpose string, ttl time.Duration) (string, error) {
	token, err := auth.GenerateRefreshTokenString() // 32 random bytes, hex
	if err != nil {
		return "", err
	}
	err = db.WithTx(ctx, s.repo.pool, func(tx pgx.Tx) error {
		return createAccountToken(ctx, tx, userID, purpose, hashToken(token), time.Now().Add(ttl))
	})
	return token, err
}

func (s *Service) link(path, token string) string {
	return fmt.Sprintf("%s/%s?token=%s", s.appURL, path, url.QueryEscape(token))
}

func (s *Service) sendVerification(ctx context.Context, user *UserRow) error {
	token, err := s.issueAccountToken(ctx, user.ID, PurposeVerifyEmail, verifyEmailTTL)
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, mail.Message{
		To:      user.Email,
		Subject: "Confirm your email for Lewe",
		Body: fmt.Sprintf("Hi %s,\n\nConfirm your email address by opening this link:\n%s\n\n"+
			"The link expires in 48 hours. If you didn't create a Lewe account, you can ignore this email.\n",
			user.FullName, s.link("verify-email", token)),
	})
}

// ResendVerification emails a new verification link to the signed-in user.
func (s *Service) ResendVerification(ctx context.Context, userID pgtype.UUID) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user.EmailVerifiedAt.Valid {
		return ErrAlreadyVerified
	}
	return s.sendVerification(ctx, user)
}

// VerifyEmail confirms the address behind an emailed verification token.
func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidAccountToken
	}
	return db.WithTx(ctx, s.repo.pool, func(tx pgx.Tx) error {
		userID, err := consumeAccountToken(ctx, tx, PurposeVerifyEmail, hashToken(token))
		if err != nil {
			return err
		}
		return markEmailVerified(ctx, tx, userID)
	})
}

// ForgotPassword emails a reset link if the address belongs to an account.
// It reports success either way so callers can't probe which emails exist.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = normalizeEmail(email)
	errs := make(validator.Errors)
	validator.ValidateEmail(errs, "email", email)
	if errs.HasErrors() {
		return errs
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if errors.Is(err, ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	token, err := s.issueAccountToken(ctx, user.ID, PurposeResetPassword, resetPasswordTTL)
	if err != nil {
		return err
	}
	err = s.mailer.Send(ctx, mail.Message{
		To:      user.Email,
		Subject: "Reset your Lewe password",
		Body: fmt.Sprintf("Hi %s,\n\nSomeone asked to reset the password for your Lewe account. "+
			"To choose a new password, open this link:\n%s\n\n"+
			"The link expires in 1 hour and works once. If you didn't ask for this, you can ignore this email; "+
			"your password stays the same.\n",
			user.FullName, s.link("reset-password", token)),
	})
	if err != nil {
		// Don't reveal the failure (or the account) to the caller.
		log.Printf("send password reset email to user %s: %v", user.ID.String(), err)
	}
	return nil
}

// ResetPassword sets a new password using an emailed reset token. It signs
// the user out everywhere, and confirms their email since they proved they
// can read it.
func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	errs := make(validator.Errors)
	validatePassword(errs, "password", req.Password)
	if errs.HasErrors() {
		return errs
	}
	if req.Token == "" {
		return ErrInvalidAccountToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.WithTx(ctx, s.repo.pool, func(tx pgx.Tx) error {
		userID, err := consumeAccountToken(ctx, tx, PurposeResetPassword, hashToken(req.Token))
		if err != nil {
			return err
		}
		if err := setPassword(ctx, tx, userID, string(hash)); err != nil {
			return err
		}
		if err := markEmailVerified(ctx, tx, userID); err != nil {
			return err
		}
		return revokeAllForUser(ctx, tx, userID)
	})
}

// ChangePassword replaces a signed-in user's password after checking the
// current one. Every other session is signed out; the caller gets a fresh
// token pair so they stay signed in.
func (s *Service) ChangePassword(ctx context.Context, userID pgtype.UUID, req ChangePasswordRequest) (*AuthResponse, error) {
	errs := make(validator.Errors)
	validator.ValidateRequired(errs, "current_password", req.CurrentPassword)
	validatePassword(errs, "new_password", req.NewPassword)
	if errs.HasErrors() {
		return nil, errs
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return nil, validator.Errors{"current_password": "is incorrect"}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	err = db.WithTx(ctx, s.repo.pool, func(tx pgx.Tx) error {
		if err := setPassword(ctx, tx, userID, string(hash)); err != nil {
			return err
		}
		return revokeAllForUser(ctx, tx, userID)
	})
	if err != nil {
		return nil, err
	}
	return s.generateAuthResponse(ctx, user)
}
