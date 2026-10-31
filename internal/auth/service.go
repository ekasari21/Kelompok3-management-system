package auth

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrInvalidToken       = errors.New("token tidak valid atau kedaluwarsa")
	ErrInvalidRole        = ("Role tidak berhak mengakses halaman")
	ErrDuplicateEmail     = errors.New("email sudah digunakan")
	ErrInvalidUser        = errors.New("data user tidak valid")
	ErrUserNotFound       = errors.New("user tidak ditemukan")
)

type Service struct {
	repo   *Repository
	secret []byte
}

// Claims menyimpan informasi user di dalam JWT.
type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func NewService(repo *Repository, secret []byte) *Service {
	return &Service{repo: repo, secret: secret}
}

func (s *Service) Register(ctx context.Context, name, email, password, role string) (User, error) {
	existingUser, err := s.repo.GetByEmail(ctx, strings.ToLower(email))
	if err == nil && existingUser.ID != 0 {
		return User{}, ErrDuplicateEmail
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	user := User{Name: strings.TrimSpace(name), Email: strings.ToLower(email), Password: string(hash), Role: strings.ToLower(role)}
	if err := s.repo.Create(ctx, &user); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) Create(
	ctx context.Context,
	input CreateUserInput,
) (*User, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Role = strings.ToLower(strings.TrimSpace(input.Role))

	if input.Name == "" || input.Email == "" ||
		len(input.Password) < 8 || len(input.Password) > 72 ||
		(input.Role != "admin" &&
			input.Role != "teknisi" &&
			input.Role != "user") {
		return nil, ErrInvalidUser
	}

	_, err := s.repo.GetByEmail(ctx, input.Email)
	if err == nil {
		return nil, ErrDuplicateEmail
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	item := &User{
		Name:      input.Name,
		Email:     input.Email,
		Password:  string(hashedPassword),
		Role:      input.Role,
		CreatedAt: now,
		UpdateAt:  now,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.GetByEmail(ctx, strings.ToLower(email))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return "", ErrInvalidCredentials
	}
	now := time.Now()

	claims := Claims{
		Email: user.Email,
		Role:  user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "gin-auth",
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}

	// ponytail: JWT berlaku satu jam; tambah revocation store saat butuh logout server-side.
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(s.secret)
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func (s *Service) Authenticate(
	ctx context.Context,
	rawToken string,
) (User, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(
		rawToken,
		claims,
		func(token *jwt.Token) (any, error) {
			return s.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer("gin-auth"),
	)
	if err != nil {
		return User{}, ErrInvalidToken
	}

	id, err := strconv.ParseUint(
		claims.Subject,
		10,
		strconv.IntSize,
	)
	if err != nil || id == 0 {
		return User{}, ErrInvalidToken
	}

	if s.repo == nil || s.repo.db == nil {
		return User{
			ID:    uint(id),
			Email: claims.Email,
			Role:  claims.Role,
		}, nil
	}

	user, err := s.repo.GetByID(ctx, uint(id))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrInvalidToken
	}
	if err != nil {
		return User{}, err
	}

	// Tolak token lama jika email atau role user sudah berubah.
	if !strings.EqualFold(claims.Email, user.Email) ||
		!strings.EqualFold(claims.Role, user.Role) {
		return User{}, ErrInvalidToken
	}

	return user, nil
}

func (s *Service) List(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(
	ctx context.Context,
	id uint,
) (*User, error) {
	item, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	return &item, err
}

func (s *Service) Update(
	ctx context.Context,
	id uint,
	input UpdateUserInput,
) (*User, error) {
	item, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Role = strings.ToLower(strings.TrimSpace(input.Role))

	if input.Name == "" || input.Email == "" ||
		(input.Role != "admin" &&
			input.Role != "teknisi" &&
			input.Role != "user") {
		return nil, ErrInvalidUser
	}

	if input.Password != "" &&
		(len(input.Password) < 8 || len(input.Password) > 72) {
		return nil, ErrInvalidUser
	}

	existing, err := s.repo.GetByEmail(ctx, input.Email)
	if err == nil && existing.ID != id {
		return nil, ErrDuplicateEmail
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	item.Name = input.Name
	item.Email = input.Email
	item.Role = input.Role
	item.UpdateAt = time.Now()

	if input.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(input.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			return nil, err
		}
		item.Password = string(hashedPassword)
	}

	if err := s.repo.Update(ctx, &item); err != nil {
		return nil, err
	}

	return &item, nil
}

func (s *Service) Delete(
	ctx context.Context,
	id uint,
) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrUserNotFound
	}
	return err
}
