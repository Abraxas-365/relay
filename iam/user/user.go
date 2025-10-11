package user

import (
	"net/http"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/ptrx"
	"github.com/Abraxas-365/relay/iam"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// User Entity
// ============================================================================

// UserStatus define los posibles estados de un usuario
type UserStatus string

const (
	UserStatusActive    UserStatus = "ACTIVE"
	UserStatusInactive  UserStatus = "INACTIVE"
	UserStatusSuspended UserStatus = "SUSPENDED"
	UserStatusPending   UserStatus = "PENDING" // Invitado pero no completó onboarding
)

// User es la entidad rica que representa a un usuario en el sistema
type User struct {
	ID              kernel.UserID     `db:"id" json:"id"`
	TenantID        kernel.TenantID   `db:"tenant_id" json:"tenant_id"`
	Email           string            `db:"email" json:"email"`
	Name            string            `db:"name" json:"name"`
	Picture         *string           `db:"picture" json:"picture,omitempty"`
	Status          UserStatus        `db:"status" json:"status"`
	IsAdmin         bool              `db:"is_admin" json:"is_admin"`
	OAuthProvider   iam.OAuthProvider `db:"oauth_provider" json:"oauth_provider"`
	OAuthProviderID string            `db:"oauth_provider_id" json:"oauth_provider_id"`
	EmailVerified   bool              `db:"email_verified" json:"email_verified"`
	LastLoginAt     *time.Time        `db:"last_login_at" json:"last_login_at,omitempty"`
	CreatedAt       time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time         `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Domain Methods
// ============================================================================

// IsActive verifica si el usuario está activo
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// CanLogin verifica si el usuario puede iniciar sesión
func (u *User) CanLogin() bool {
	return u.IsActive() && u.EmailVerified
}

// Activate activa un usuario pendiente
func (u *User) Activate() error {
	if u.Status != UserStatusPending {
		return ErrInvalidStatus().WithDetail("current_status", u.Status)
	}

	u.Status = UserStatusActive
	u.UpdatedAt = time.Now()
	return nil
}

// Suspend suspende un usuario activo
func (u *User) Suspend(reason string) error {
	if !u.IsActive() {
		return ErrInvalidStatus().WithDetail("current_status", u.Status)
	}

	u.Status = UserStatusSuspended
	u.UpdatedAt = time.Now()
	return nil
}

// UpdateLastLogin actualiza la fecha del último login
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// UpdateProfile actualiza la información del perfil
func (u *User) UpdateProfile(name, picture string) {
	if name != "" {
		u.Name = name
	}
	if picture != "" {
		u.Picture = ptrx.String(picture)
	}
	u.UpdatedAt = time.Now()
}

// MakeAdmin convierte al usuario en administrador
func (u *User) MakeAdmin() {
	u.IsAdmin = true
	u.UpdatedAt = time.Now()
}

// RevokeAdmin remueve permisos de administrador
func (u *User) RevokeAdmin() {
	u.IsAdmin = false
	u.UpdatedAt = time.Now()
}

// ============================================================================
// DTOs
// ============================================================================

// UserDetailsDTO contiene información básica de un usuario para otros módulos
type UserDetailsDTO struct {
	ID            kernel.UserID     `json:"id"`
	TenantID      kernel.TenantID   `json:"tenant_id"`
	Name          string            `json:"name"`
	Email         string            `json:"email"`
	Picture       *string           `json:"picture,omitempty"`
	IsActive      bool              `json:"is_active"`
	IsAdmin       bool              `json:"is_admin"`
	OAuthProvider iam.OAuthProvider `json:"oauth_provider"`
}

// ToDTO convierte la entidad User a UserDetailsDTO
func (u *User) ToDTO() UserDetailsDTO {
	return UserDetailsDTO{
		ID:            u.ID,
		TenantID:      u.TenantID,
		Name:          u.Name,
		Email:         u.Email,
		Picture:       u.Picture,
		IsActive:      u.IsActive(),
		IsAdmin:       u.IsAdmin,
		OAuthProvider: u.OAuthProvider,
	}
}

// ============================================================================
// Service DTOs - Para operaciones de la capa de servicio
// ============================================================================

// CreateUserRequest representa la petición para crear un usuario
type CreateUserRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Email    string          `json:"email" validate:"required,email"`
	Name     string          `json:"name" validate:"required,min=2"`
	IsAdmin  bool            `json:"is_admin"`
	RoleIDs  []kernel.RoleID `json:"role_ids,omitempty"`
}

// UpdateUserRequest representa la petición para actualizar un usuario
type UpdateUserRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Name     *string         `json:"name,omitempty" validate:"omitempty,min=2"`
	Status   *UserStatus     `json:"status,omitempty"`
	IsAdmin  *bool           `json:"is_admin,omitempty"`
}

// InviteUserRequest para invitar usuarios a un tenant
type InviteUserRequest struct {
	Email   string `json:"email" validate:"required,email"`
	IsAdmin bool   `json:"is_admin"`
}

// UserResponse representa la respuesta completa de un usuario con sus roles
type UserResponse struct {
	User    User            `json:"user"`
	RoleIDs []kernel.RoleID `json:"role_ids"`
}

// ToDTO convierte UserResponse a UserResponseDTO
func (ur *UserResponse) ToDTO() UserResponseDTO {
	return UserResponseDTO{
		User:    ur.User.ToDTO(),
		RoleIDs: ur.RoleIDs,
	}
}

// UserResponseDTO es la versión DTO de UserResponse
type UserResponseDTO struct {
	User    UserDetailsDTO  `json:"user"`
	RoleIDs []kernel.RoleID `json:"role_ids"`
}

// AssignRoleRequest para asignar un rol a un usuario
type AssignRoleRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	RoleID   kernel.RoleID   `json:"role_id" validate:"required"`
}

// RemoveRoleRequest para remover un rol de un usuario
type RemoveRoleRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	RoleID   kernel.RoleID   `json:"role_id" validate:"required"`
}

// SuspendUserRequest para suspender un usuario
type SuspendUserRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Reason   string          `json:"reason" validate:"required,min=5"`
}

// ActivateUserRequest para activar un usuario
type ActivateUserRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
}

// UserListResponse para listas de usuarios
type UserListResponse struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
}

// ToDTO convierte UserListResponse a UserListResponseDTO
func (ulr *UserListResponse) ToDTO() UserListResponseDTO {
	var usersDTO []UserResponseDTO
	for _, u := range ulr.Users {
		usersDTO = append(usersDTO, u.ToDTO())
	}

	return UserListResponseDTO{
		Users: usersDTO,
		Total: ulr.Total,
	}
}

// UserListResponseDTO es la versión DTO de UserListResponse
type UserListResponseDTO struct {
	Users []UserResponseDTO `json:"users"`
	Total int               `json:"total"`
}

// ============================================================================
// Error Registry - Errores específicos de User
// ============================================================================

var ErrRegistry = errx.NewRegistry("USER")

// Códigos de error
var (
	CodeUserNotFound       = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Usuario no encontrado")
	CodeUserAlreadyExists  = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "El usuario ya existe")
	CodeUserNotInTenant    = ErrRegistry.Register("NOT_IN_TENANT", errx.TypeAuthorization, http.StatusForbidden, "Usuario no pertenece a la empresa")
	CodeEmailNotVerified   = ErrRegistry.Register("EMAIL_NOT_VERIFIED", errx.TypeBusiness, http.StatusPreconditionFailed, "Email no verificado")
	CodeUserSuspended      = ErrRegistry.Register("SUSPENDED", errx.TypeBusiness, http.StatusForbidden, "Usuario suspendido")
	CodeOnboardingRequired = ErrRegistry.Register("ONBOARDING_REQUIRED", errx.TypeBusiness, http.StatusPreconditionRequired, "Se requiere completar el onboarding")
	CodeInvalidStatus      = ErrRegistry.Register("INVALID_STATUS", errx.TypeBusiness, http.StatusBadRequest, "Estado de usuario inválido para esta operación")
)

// Helper functions para crear errores
func ErrUserNotFound() *errx.Error {
	return ErrRegistry.New(CodeUserNotFound)
}

func ErrUserAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeUserAlreadyExists)
}

func ErrUserNotInTenant() *errx.Error {
	return ErrRegistry.New(CodeUserNotInTenant)
}

func ErrEmailNotVerified() *errx.Error {
	return ErrRegistry.New(CodeEmailNotVerified)
}

func ErrUserSuspended() *errx.Error {
	return ErrRegistry.New(CodeUserSuspended)
}

func ErrOnboardingRequired() *errx.Error {
	return ErrRegistry.New(CodeOnboardingRequired)
}

func ErrInvalidStatus() *errx.Error {
	return ErrRegistry.New(CodeInvalidStatus)
}
