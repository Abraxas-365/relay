package kernel

// ============================================================================
// Context Types - Tipos para context.Context
// ============================================================================

// AuthContext es el contexto de autenticación que se inyecta en cada request
type AuthContext struct {
	UserID   UserID   `json:"user_id"`
	TenantID TenantID `json:"tenant_id"`
	IsAdmin  bool     `json:"is_admin"`
	Email    string   `json:"email"`
	Name     string   `json:"name"`
}

// IsValid verifica si el AuthContext es válido
func (a *AuthContext) IsValid() bool {
	return !a.UserID.IsEmpty() && !a.TenantID.IsEmpty()
}

// ============================================================================
// Context Keys - Claves para context.Context
// ============================================================================

type ContextKey string

const (
	// AuthContextKey es la clave para almacenar AuthContext en context.Context
	AuthContextKey ContextKey = "auth_context"

	// TenantContextKey es la clave para almacenar TenantID en context.Context
	TenantContextKey ContextKey = "tenant_id"

	// UserContextKey es la clave para almacenar UserID en context.Context
	UserContextKey ContextKey = "user_id"

	// RequestIDKey es la clave para almacenar el ID de la petición
	RequestIDKey ContextKey = "request_id"
)
