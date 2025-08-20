package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Middleware provides authentication middleware for Docker Registry requests
type Middleware struct {
	realm   string
	service string
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(realm, service string) *Middleware {
	return &Middleware{
		realm:   realm,
		service: service,
	}
}

// DockerRegistryAuth is a middleware that handles Docker Registry authentication
func (m *Middleware) DockerRegistryAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for the /v2/ endpoint (API version check)
		if r.URL.Path == "/v2/" {
			next.ServeHTTP(w, r)
			return
		}

		// Check for Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No authorization header, return 401 with WWW-Authenticate header
			m.challengeAuth(w, r)
			return
		}

		// Check if this is a Bearer token
		if strings.HasPrefix(authHeader, "Bearer ") {
			m.handleBearerAuth(w, r, next)
			return
		}

		// Handle Basic Auth (existing flow)
		m.handleBasicAuth(w, r, next)
	})
}

// handleBearerAuth processes Bearer token authentication
func (m *Middleware) handleBearerAuth(w http.ResponseWriter, r *http.Request, next http.Handler) {
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	
	if token == "" {
		m.challengeAuth(w, r)
		return
	}

	// For Bearer tokens, we need to extract the registry URL from somewhere
	// We'll use a default or try to extract from request headers/cookies
	registryURL := m.extractRegistryURL(r)
	if registryURL == "" {
		// Default to Docker Hub if we can't determine the registry
		registryURL = "registry-1.docker.io"
	}

	// Create bearer auth context
	bearerAuth := &BearerAuth{
		Token:       token,
		RegistryURL: registryURL,
	}

	// Add bearer auth context to request
	ctx := context.WithValue(r.Context(), "bearer", bearerAuth)
	r = r.WithContext(ctx)

	// Continue to next handler
	next.ServeHTTP(w, r)
}

// handleBasicAuth processes Basic authentication (existing flow)
func (m *Middleware) handleBasicAuth(w http.ResponseWriter, r *http.Request, next http.Handler) {
	// Parse Basic Auth
	username, password, ok := r.BasicAuth()
	if !ok {
		m.challengeAuth(w, r)
		return
	}

	// Validate username format
	_, err := ParseUsername(username)
	if err != nil {
		m.writeErrorResponse(w, "UNAUTHORIZED", "Invalid username format", http.StatusUnauthorized)
		return
	}

	// Add auth context to request
	authCtx := &AuthHeader{
		Username: username,
		Password: password,
	}

	// Add auth context to request context
	ctx := context.WithValue(r.Context(), "auth", authCtx)
	r = r.WithContext(ctx)

	// Continue to next handler
	next.ServeHTTP(w, r)
}

// extractRegistryURL attempts to extract the registry URL for Bearer token requests
func (m *Middleware) extractRegistryURL(r *http.Request) string {
	// Try to get from custom header (if Aqua sets it)
	if registryURL := r.Header.Get("X-Registry-URL"); registryURL != "" {
		return registryURL
	}
	
	// Try to get from cookies (if we set it during Basic Auth)
	if cookie, err := r.Cookie("registry-url"); err == nil {
		return cookie.Value
	}
	
	// Default to Docker Hub registry
	return "registry-1.docker.io"
}

// challengeAuth returns a 401 Unauthorized response with WWW-Authenticate header
func (m *Middleware) challengeAuth(w http.ResponseWriter, r *http.Request) {
	// Extract scope from request path for more specific authentication challenge
	scope := m.extractScope(r)

	authHeader := fmt.Sprintf(`Bearer realm="%s",service="%s"`, m.realm, m.service)
	if scope != "" {
		authHeader += fmt.Sprintf(`,scope="%s"`, scope)
	}

	w.Header().Set("WWW-Authenticate", authHeader)
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")

	m.writeErrorResponse(w, "UNAUTHORIZED", "authentication required", http.StatusUnauthorized)
}

// extractScope extracts the scope from the request path for authentication challenge
func (m *Middleware) extractScope(r *http.Request) string {
	path := r.URL.Path

	// Extract repository name and action from path
	if strings.HasPrefix(path, "/v2/") {
		path = strings.TrimPrefix(path, "/v2/")
		
		// Handle catalog request
		if path == "_catalog" {
			return "registry:catalog:*"
		}

		// Handle repository-specific requests
		if strings.Contains(path, "/") {
			parts := strings.Split(path, "/")
			if len(parts) >= 2 {
				repoName := strings.Join(parts[:len(parts)-1], "/")
				endpoint := parts[len(parts)-1]

				// Determine action based on endpoint and HTTP method
				action := m.getActionFromEndpoint(endpoint, r.Method)
				return fmt.Sprintf("repository:%s:%s", repoName, action)
			}
		}
	}

	return ""
}

// getActionFromEndpoint determines the action based on endpoint and HTTP method
func (m *Middleware) getActionFromEndpoint(endpoint, method string) string {
	switch {
	case endpoint == "tags" || strings.HasPrefix(endpoint, "tags/"):
		return "pull"
	case strings.HasPrefix(endpoint, "manifests/"):
		switch method {
		case "GET":
			return "pull"
		case "PUT":
			return "push"
		case "DELETE":
			return "delete"
		default:
			return "pull"
		}
	case strings.HasPrefix(endpoint, "blobs/"):
		switch method {
		case "GET":
			return "pull"
		case "DELETE":
			return "delete"
		default:
			return "pull"
		}
	case endpoint == "uploads" || strings.HasPrefix(endpoint, "uploads/"):
		return "push"
	default:
		return "pull"
	}
}

// writeErrorResponse writes a Docker Registry API compliant error response
func (m *Middleware) writeErrorResponse(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]interface{}{
		"errors": []map[string]interface{}{
			{
				"code":    code,
				"message": message,
			},
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}

// GetAuthFromContext extracts authentication information from request context
func GetAuthFromContext(ctx context.Context) (*AuthHeader, bool) {
	auth, ok := ctx.Value("auth").(*AuthHeader)
	return auth, ok
}

// GetBearerAuthFromContext extracts bearer authentication information from request context
func GetBearerAuthFromContext(ctx context.Context) (*BearerAuth, bool) {
	bearerAuth, ok := ctx.Value("bearer").(*BearerAuth)
	return bearerAuth, ok
}

// RequireAuth is a helper function to extract and validate auth from request
func RequireAuth(r *http.Request) (*AuthHeader, error) {
	auth, ok := GetAuthFromContext(r.Context())
	if !ok {
		return nil, fmt.Errorf("authentication required")
	}

	if auth.Username == "" || auth.Password == "" {
		return nil, fmt.Errorf("invalid authentication credentials")
	}

	return auth, nil
}