package auth

import (
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/golang-jwt/jwt/v5"
)

// JWTService implementación del TokenService usando JWT
type JWTService struct {
	secretKey       []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

// NewJWTService crea una nueva instancia del servicio JWT
func NewJWTService(secretKey string, accessTokenTTL, refreshTokenTTL time.Duration, issuer string) *JWTService {
	if accessTokenTTL == 0 {
		accessTokenTTL = 15 * time.Minute // Por defecto 15 minutos
	}
	if refreshTokenTTL == 0 {
		refreshTokenTTL = 7 * 24 * time.Hour // Por defecto 7 días
	}
	if issuer == "" {
		issuer = "facturamelo"
	}

	return &JWTService{
		secretKey:       []byte(secretKey),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
		issuer:          issuer,
	}
}

// Claims personalizados para JWT
type JWTClaims struct {
	UserID   kernel.UserID   `json:"user_id"`
	TenantID kernel.TenantID `json:"tenant_id"`
	Email    string          `json:"email"`
	Name     string          `json:"name"`
	IsAdmin  bool            `json:"is_admin"`
	jwt.RegisteredClaims
}

// GenerateAccessToken genera un token de acceso JWT
func (j *JWTService) GenerateAccessToken(userID kernel.UserID, tenantID kernel.TenantID, claims map[string]any) (string, error) {
	now := time.Now()

	// Extraer claims adicionales
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	isAdmin, _ := claims["is_admin"].(bool)

	jwtClaims := JWTClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Name:     name,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID.String(),
			Audience:  []string{"facturamelo-api"},
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)

	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", ErrTokenGenerationFailed().WithDetail("error", err.Error())
	}

	return tokenString, nil
}

// ValidateAccessToken valida y decodifica un token de acceso
func (j *JWTService) ValidateAccessToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
		// Verificar el método de firma
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, ErrTokenValidationFailed().WithDetail("error", err.Error())
	}

	if !token.Valid {
		return nil, ErrTokenValidationFailed().WithDetail("error", "token is invalid")
	}

	jwtClaims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, ErrTokenValidationFailed().WithDetail("error", "invalid claims type")
	}

	return &TokenClaims{
		UserID:    jwtClaims.UserID,
		TenantID:  jwtClaims.TenantID,
		Email:     jwtClaims.Email,
		Name:      jwtClaims.Name,
		IsAdmin:   jwtClaims.IsAdmin,
		IssuedAt:  jwtClaims.IssuedAt.Time,
		ExpiresAt: jwtClaims.ExpiresAt.Time,
	}, nil
}

// GenerateRefreshToken genera un token de refresh simple
func (j *JWTService) GenerateRefreshToken(userID kernel.UserID) (string, error) {
	now := time.Now()

	claims := jwt.RegisteredClaims{
		Issuer:    j.issuer,
		Subject:   userID.String(),
		Audience:  []string{"facturamelo-refresh"},
		ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTokenTTL)),
		NotBefore: jwt.NewNumericDate(now),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", ErrTokenGenerationFailed().WithDetail("error", err.Error())
	}

	return tokenString, nil
}
