package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ayam-batu-api/internal/middleware"
	"ayam-batu-api/internal/models"
	"ayam-batu-api/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	ErrEmailAlreadyUsed      = errors.New("email is already registered")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrGoogleAccountOnly     = errors.New("this account uses Google login; please sign in via Google")
	ErrGoogleOAuthNotEnabled = errors.New("google oauth is not configured; set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET in .env")
	ErrInvalidOAuthState     = errors.New("invalid or expired oauth state")
	ErrGoogleEmailUnverified = errors.New("google account email is missing or not verified")
)

const googleUserInfoEndpoint = "https://openidconnect.googleapis.com/v1/userinfo"

// GoogleOAuthOptions configures Google OAuth 2.0 redirect login.
type GoogleOAuthOptions struct {
	ClientID            string
	ClientSecret        string
	RedirectURL         string
	FrontendRedirectURL string
	Endpoint            oauth2.Endpoint
	UserInfoURL         string
}

type googleUserInfo struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

type oauthStateClaims struct {
	Nonce string `json:"nonce"`
	jwt.RegisteredClaims
}

// AuthService manages user registration, login, Google OAuth, and JWT generation.
type AuthService struct {
	userRepo        *repository.UserRepository
	jwtSecret       string
	jwtExpiresHours int
	googleOpts      GoogleOAuthOptions
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpiresHours int, googleOpts ...GoogleOAuthOptions) *AuthService {
	var opts GoogleOAuthOptions
	if len(googleOpts) > 0 {
		opts = googleOpts[0]
	}
	if opts.Endpoint.AuthURL == "" {
		opts.Endpoint = google.Endpoint
	}
	if opts.UserInfoURL == "" {
		opts.UserInfoURL = googleUserInfoEndpoint
	}
	return &AuthService{
		userRepo:        userRepo,
		jwtSecret:       jwtSecret,
		jwtExpiresHours: jwtExpiresHours,
		googleOpts:      opts,
	}
}

// SignUpInput represents the payload for user registration.
type SignUpInput struct {
	Name     string `json:"name" binding:"required,min=2,max=120"`
	Email    string `json:"email" binding:"required,email,max=180"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

// LoginInput represents the payload for user login.
type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResult contains the issued JWT token and user profile.
type AuthResult struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func (s *AuthService) SignUp(input SignUpInput) (*AuthResult, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	name := strings.TrimSpace(input.Name)

	existing, err := s.userRepo.FindByEmail(email)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailAlreadyUsed
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hash)

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: &hashStr,
		AuthProvider: "local",
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, User: user}, nil
}

func (s *AuthService) Login(input LoginInput) (*AuthResult, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))
	user, err := s.userRepo.FindByEmail(email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		return nil, ErrGoogleAccountOnly
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{Token: token, User: user}, nil
}

func (s *AuthService) GetProfile(userID uint) (*models.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *AuthService) IsGoogleOAuthConfigured() bool {
	return strings.TrimSpace(s.googleOpts.ClientID) != "" && strings.TrimSpace(s.googleOpts.ClientSecret) != ""
}

func (s *AuthService) GenerateOAuthState() (string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", err
	}
	now := time.Now()
	claims := oauthStateClaims{
		Nonce: hex.EncodeToString(nonceBytes),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) ValidateOAuthState(queryState, cookieState string) error {
	if queryState == "" || cookieState == "" {
		return ErrInvalidOAuthState
	}
	if subtle.ConstantTimeCompare([]byte(queryState), []byte(cookieState)) != 1 {
		return ErrInvalidOAuthState
	}

	claims := &oauthStateClaims{}
	token, err := jwt.ParseWithClaims(queryState, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid || claims.Nonce == "" {
		return ErrInvalidOAuthState
	}
	return nil
}

func (s *AuthService) GoogleAuthURL(state string) (string, error) {
	if !s.IsGoogleOAuthConfigured() {
		return "", ErrGoogleOAuthNotEnabled
	}
	return s.oauthConfig().AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

func (s *AuthService) HandleGoogleCallback(ctx context.Context, code string) (*AuthResult, string, error) {
	if !s.IsGoogleOAuthConfigured() {
		return nil, "", ErrGoogleOAuthNotEnabled
	}

	cfg := s.oauthConfig()
	oauthToken, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("exchange google oauth code: %w", err)
	}

	info, err := s.fetchGoogleUserInfo(ctx, cfg.Client(ctx, oauthToken))
	if err != nil {
		return nil, "", err
	}

	email := strings.TrimSpace(strings.ToLower(info.Email))
	if info.Sub == "" || email == "" || !info.EmailVerified {
		return nil, "", ErrGoogleEmailUnverified
	}

	user, err := s.upsertGoogleUser(info.Sub, email, strings.TrimSpace(info.Name))
	if err != nil {
		return nil, "", err
	}

	jwtToken, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return &AuthResult{Token: jwtToken, User: user}, strings.TrimSpace(s.googleOpts.FrontendRedirectURL), nil
}

func (s *AuthService) oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.googleOpts.ClientID,
		ClientSecret: s.googleOpts.ClientSecret,
		RedirectURL:  s.googleOpts.RedirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     s.googleOpts.Endpoint,
	}
}

func (s *AuthService) fetchGoogleUserInfo(ctx context.Context, client *http.Client) (*googleUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.googleOpts.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch google userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("google userinfo returned %d: %s", resp.StatusCode, string(body))
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode google userinfo: %w", err)
	}
	return &info, nil
}

func (s *AuthService) upsertGoogleUser(googleID, email, name string) (*models.User, error) {
	user, err := s.userRepo.FindByGoogleID(googleID)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	existingByEmail, err := s.userRepo.FindByEmail(email)
	if err == nil {
		existingByEmail.GoogleID = &googleID
		if existingByEmail.Name == "" && name != "" {
			existingByEmail.Name = name
		}
		if err := s.userRepo.Update(existingByEmail); err != nil {
			return nil, err
		}
		return existingByEmail, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	newUser := &models.User{
		Name:         name,
		Email:        email,
		AuthProvider: "google",
		GoogleID:     &googleID,
	}
	if err := s.userRepo.Create(newUser); err != nil {
		return nil, err
	}
	return newUser, nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	now := time.Now()
	claims := middleware.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.jwtExpiresHours) * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
