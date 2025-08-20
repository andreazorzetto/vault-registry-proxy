package auth

import (
	"errors"
	"strings"
)

var (
	ErrInvalidUsernameFormat = errors.New("invalid username format, expected: <registry_type>;<vault_path>;<registry_url>")
	ErrUnsupportedRegistryType = errors.New("unsupported registry type")
)

// RegistryConfig represents the parsed configuration from the username field
type RegistryConfig struct {
	Type        string // e.g., "docker", "ecr", "gcr"
	VaultPath   string // path in Vault KV store
	RegistryURL string // actual registry URL
}

// ParseUsername parses the username field format: <registry_type>;<vault_path>;<registry_url>
// Example: "docker;secret/docker-hub;registry.hub.docker.com"
func ParseUsername(username string) (*RegistryConfig, error) {
	parts := strings.SplitN(username, ";", 3)
	if len(parts) != 3 {
		return nil, ErrInvalidUsernameFormat
	}

	registryType := strings.TrimSpace(parts[0])
	vaultPath := strings.TrimSpace(parts[1])
	registryURL := strings.TrimSpace(parts[2])

	if registryType == "" || vaultPath == "" || registryURL == "" {
		return nil, ErrInvalidUsernameFormat
	}

	// Validate supported registry types
	if !isValidRegistryType(registryType) {
		return nil, ErrUnsupportedRegistryType
	}

	return &RegistryConfig{
		Type:        registryType,
		VaultPath:   vaultPath,
		RegistryURL: registryURL,
	}, nil
}

// isValidRegistryType checks if the registry type is supported
func isValidRegistryType(registryType string) bool {
	supportedTypes := map[string]bool{
		"docker": true,
		"ecr":    true,
		"gcr":    true,
	}
	return supportedTypes[registryType]
}

// Credentials represents the actual registry credentials retrieved from Vault
type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email,omitempty"`
}

// AuthHeader represents authentication information from the request
type AuthHeader struct {
	Username string // The parsed username containing registry config
	Password string // The Vault token
}

// BearerAuth represents bearer token authentication information
type BearerAuth struct {
	Token       string // The bearer token
	RegistryURL string // The target registry URL (extracted from previous Basic Auth)
}

// ParseAuthHeader extracts authentication information from HTTP basic auth header
func ParseAuthHeader(username, password string) *AuthHeader {
	return &AuthHeader{
		Username: username,
		Password: password,
	}
}