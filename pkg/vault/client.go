package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"

	"vault-docker-proxy/pkg/auth"
)

var (
	ErrVaultConnection = errors.New("failed to connect to Vault")
	ErrInvalidToken    = errors.New("invalid Vault token")
	ErrSecretNotFound  = errors.New("secret not found in Vault")
)

// Client wraps the HashiCorp Vault API client
type Client struct {
	client *api.Client
	config *Config
}

// Config holds Vault client configuration
type Config struct {
	Address string
	Token   string
}

// NewClient creates a new Vault client
func NewClient(vaultAddr string) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = vaultAddr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVaultConnection, err)
	}

	return &Client{
		client: client,
		config: &Config{
			Address: vaultAddr,
		},
	}, nil
}

// SetToken sets the Vault token for authentication
func (c *Client) SetToken(token string) {
	c.client.SetToken(token)
	c.config.Token = token
}

// GetCredentials retrieves registry credentials from Vault KV store
func (c *Client) GetCredentials(ctx context.Context, vaultPath string) (*auth.Credentials, error) {
	// Use KV v2 secrets engine
	secret, err := c.client.KVv2("secret").Get(ctx, vaultPath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSecretNotFound, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, ErrSecretNotFound
	}

	// Extract credentials from secret data
	data := secret.Data
	username, ok := data["username"].(string)
	if !ok {
		return nil, errors.New("username not found in secret")
	}

	password, ok := data["password"].(string)
	if !ok {
		return nil, errors.New("password not found in secret")
	}

	// Email is optional
	email := ""
	if emailValue, exists := data["email"]; exists {
		if emailStr, ok := emailValue.(string); ok {
			email = emailStr
		}
	}

	return &auth.Credentials{
		Username: username,
		Password: password,
		Email:    email,
	}, nil
}

// ValidateToken checks if the current token is valid
func (c *Client) ValidateToken(ctx context.Context) error {
	if c.config.Token == "" {
		return ErrInvalidToken
	}

	// Check token by attempting to read token self information
	auth := c.client.Auth()
	tokenInfo, err := auth.Token().LookupSelf()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if tokenInfo == nil {
		return ErrInvalidToken
	}

	return nil
}

// Close cleans up the Vault client resources
func (c *Client) Close() error {
	// HashiCorp Vault client doesn't require explicit cleanup
	return nil
}