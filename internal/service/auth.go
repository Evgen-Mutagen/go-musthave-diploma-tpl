package service

import (
	"context"
	"errors"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/model"
	"github.com/Evgen-Mutagen/go-musthave-diploma-tpl/internal/repository"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type AuthService interface {
	Register(ctx context.Context, login, password string) (*model.User, string, error)
	Login(ctx context.Context, login, password string) (*model.User, string, error)
	ValidateToken(tokenString string) (int64, error)
}

type authService struct {
	userRepo     repository.UserRepository
	jwtSecretKey string
}

func NewAuthService(userRepo repository.UserRepository) AuthService {
	return &authService{
		userRepo:     userRepo,
		jwtSecretKey: "secret-key", // В реальном приложении должно быть из конфига
	}
}

func (s *authService) Register(ctx context.Context, login, password string) (*model.User, string, error) {
	existingUser, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		return nil, "", err
	}
	if existingUser != nil {
		return nil, "", ErrUserAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := &model.User{
		Login:        login,
		PasswordHash: string(hashedPassword),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) Login(ctx context.Context, login, password string) (*model.User, string, error) {
	user, err := s.userRepo.GetByLogin(ctx, login)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *authService) ValidateToken(tokenString string) (int64, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecretKey), nil
	})
	if err != nil {
		return 0, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID := int64(claims["user_id"].(float64))
		return userID, nil
	}

	return 0, jwt.ErrSignatureInvalid
}

func (s *authService) generateToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecretKey))
}
