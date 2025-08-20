package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"vault-docker-proxy/pkg/auth"
	"vault-docker-proxy/pkg/cache"
	"vault-docker-proxy/pkg/vault"
)

// ProxyServer handles Docker Registry v2 API requests and forwards them to the actual registry
type ProxyServer struct {
	vaultClient *vault.Client
	cache       *cache.CredentialCache
	httpClient  *http.Client
}

// NewProxyServer creates a new registry proxy server
func NewProxyServer(vaultClient *vault.Client) *ProxyServer {
	return &ProxyServer{
		vaultClient: vaultClient,
		cache:       cache.NewCredentialCache(),
		httpClient:  &http.Client{},
	}
}

// APIVersionCheck handles GET /v2/ - Docker Registry API version check
func (p *ProxyServer) APIVersionCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	w.WriteHeader(http.StatusOK)
}

// GetCatalog handles GET /v2/_catalog - retrieve repository catalog
func (p *ProxyServer) GetCatalog(w http.ResponseWriter, r *http.Request) {
	log.Printf("GetCatalog request from %s", r.RemoteAddr)
	
	// Check if this is a Bearer token request
	if bearerAuth, ok := auth.GetBearerAuthFromContext(r.Context()); ok {
		log.Printf("Using Bearer token for catalog request to registry: %s", bearerAuth.RegistryURL)
		err := p.proxyBearerRequest(w, r, bearerAuth, "/_catalog")
		if err != nil {
			log.Printf("Failed to proxy Bearer catalog request: %v", err)
			http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully proxied Bearer catalog request")
		return
	}

	// Handle Basic Auth (existing flow)
	credentials, registryConfig, err := p.authenticateAndGetCredentials(r)
	if err != nil {
		log.Printf("Authentication failed for catalog request: %v", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	log.Printf("Proxying catalog request to registry: %s", registryConfig.RegistryURL)
	
	// Forward request to actual registry
	err = p.proxyRequest(w, r, credentials, registryConfig, "/_catalog")
	if err != nil {
		log.Printf("Failed to proxy catalog request: %v", err)
		http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Successfully proxied catalog request")
}

// GetTags handles GET /v2/{name}/tags/list - fetch tags for a repository
func (p *ProxyServer) GetTags(w http.ResponseWriter, r *http.Request) {
	// Extract repository name from path
	path := r.URL.Path
	repoPath := strings.TrimPrefix(path, "/v2/")
	repoPath = strings.TrimSuffix(repoPath, "/tags/list")
	targetPath := fmt.Sprintf("/%s/tags/list", repoPath)

	// Check if this is a Bearer token request
	if bearerAuth, ok := auth.GetBearerAuthFromContext(r.Context()); ok {
		log.Printf("Using Bearer token for tags request to registry: %s, repo: %s", bearerAuth.RegistryURL, repoPath)
		err := p.proxyBearerRequest(w, r, bearerAuth, targetPath)
		if err != nil {
			log.Printf("Failed to proxy Bearer tags request: %v", err)
			http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully proxied Bearer tags request for repo: %s", repoPath)
		return
	}

	// Handle Basic Auth (existing flow)
	credentials, registryConfig, err := p.authenticateAndGetCredentials(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	err = p.proxyRequest(w, r, credentials, registryConfig, targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
		return
	}
}

// GetManifest handles GET /v2/{name}/manifests/{reference} - retrieve manifest
func (p *ProxyServer) GetManifest(w http.ResponseWriter, r *http.Request) {
	// Check if this is a Bearer token request
	if bearerAuth, ok := auth.GetBearerAuthFromContext(r.Context()); ok {
		path := strings.TrimPrefix(r.URL.Path, "/v2")
		log.Printf("Using Bearer token for manifest request to registry: %s, path: %s", bearerAuth.RegistryURL, path)
		err := p.proxyBearerRequest(w, r, bearerAuth, path)
		if err != nil {
			log.Printf("Failed to proxy Bearer manifest request: %v", err)
			http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully proxied Bearer manifest request for path: %s", path)
		return
	}

	// Handle Basic Auth (existing flow)
	credentials, registryConfig, err := p.authenticateAndGetCredentials(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Extract path from original request
	path := strings.TrimPrefix(r.URL.Path, "/v2")
	err = p.proxyRequest(w, r, credentials, registryConfig, path)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
		return
	}
}

// GetBlob handles GET /v2/{name}/blobs/{digest} - retrieve blob
func (p *ProxyServer) GetBlob(w http.ResponseWriter, r *http.Request) {
	// Check if this is a Bearer token request
	if bearerAuth, ok := auth.GetBearerAuthFromContext(r.Context()); ok {
		path := strings.TrimPrefix(r.URL.Path, "/v2")
		log.Printf("Using Bearer token for blob request to registry: %s, path: %s", bearerAuth.RegistryURL, path)
		err := p.proxyBearerRequest(w, r, bearerAuth, path)
		if err != nil {
			log.Printf("Failed to proxy Bearer blob request: %v", err)
			http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
			return
		}
		log.Printf("Successfully proxied Bearer blob request for path: %s", path)
		return
	}

	// Handle Basic Auth (existing flow)
	credentials, registryConfig, err := p.authenticateAndGetCredentials(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Extract path from original request
	path := strings.TrimPrefix(r.URL.Path, "/v2")
	err = p.proxyRequest(w, r, credentials, registryConfig, path)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
		return
	}
}

// authenticateAndGetCredentials extracts auth info and retrieves credentials from Vault
func (p *ProxyServer) authenticateAndGetCredentials(r *http.Request) (*auth.Credentials, *auth.RegistryConfig, error) {
	// Extract Basic Auth from request
	username, password, ok := r.BasicAuth()
	if !ok {
		log.Printf("Request missing basic authentication from %s", r.RemoteAddr)
		return nil, nil, fmt.Errorf("basic authentication required")
	}

	// Parse username to get registry configuration
	registryConfig, err := auth.ParseUsername(username)
	if err != nil {
		log.Printf("Invalid username format: %s, error: %v", username, err)
		return nil, nil, fmt.Errorf("invalid username format: %v", err)
	}

	log.Printf("Authenticating for registry: %s, vault path: %s", registryConfig.RegistryURL, registryConfig.VaultPath)

	// Set Vault token from password field
	p.vaultClient.SetToken(password)

	// Check cache first
	if credentials, found := p.cache.Get(password, registryConfig.VaultPath); found {
		log.Printf("Using cached credentials for path: %s", registryConfig.VaultPath)
		return credentials, registryConfig, nil
	}

	log.Printf("Retrieving credentials from Vault for path: %s", registryConfig.VaultPath)

	// Get credentials from Vault
	credentials, err := p.vaultClient.GetCredentials(context.Background(), registryConfig.VaultPath)
	if err != nil {
		log.Printf("Failed to retrieve credentials from Vault for path %s: %v", registryConfig.VaultPath, err)
		return nil, nil, fmt.Errorf("failed to retrieve credentials from Vault: %v", err)
	}

	log.Printf("Successfully retrieved credentials from Vault for path: %s", registryConfig.VaultPath)

	// Cache the credentials
	p.cache.Set(password, registryConfig.VaultPath, credentials)

	return credentials, registryConfig, nil
}

// proxyBearerRequest forwards Bearer token requests directly to the registry
func (p *ProxyServer) proxyBearerRequest(w http.ResponseWriter, r *http.Request, bearerAuth *auth.BearerAuth, targetPath string) error {
	// Build target URL
	registryURL := bearerAuth.RegistryURL
	if !strings.HasPrefix(registryURL, "http://") && !strings.HasPrefix(registryURL, "https://") {
		registryURL = "https://" + registryURL
	}

	targetURL := fmt.Sprintf("%s/v2%s", registryURL, targetPath)

	// Add query parameters
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create new request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %v", err)
	}

	// Copy headers (including the original Authorization Bearer token)
	for name, values := range r.Header {
		proxyReq.Header[name] = values
	}

	// Forward request
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("failed to forward request: %v", err)
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		w.Header()[name] = values
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body: %v", err)
	}

	return nil
}

// proxyRequest forwards the request to the actual Docker registry
func (p *ProxyServer) proxyRequest(w http.ResponseWriter, r *http.Request, credentials *auth.Credentials, registryConfig *auth.RegistryConfig, targetPath string) error {
	// Build target URL
	registryURL := registryConfig.RegistryURL
	if !strings.HasPrefix(registryURL, "http://") && !strings.HasPrefix(registryURL, "https://") {
		registryURL = "https://" + registryURL
	}

	targetURL := fmt.Sprintf("%s/v2%s", registryURL, targetPath)

	// Add query parameters
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Create new request
	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %v", err)
	}

	// Copy headers (excluding Authorization which we'll replace)
	for name, values := range r.Header {
		if name != "Authorization" {
			proxyReq.Header[name] = values
		}
	}

	// Set authentication with actual registry credentials
	proxyReq.SetBasicAuth(credentials.Username, credentials.Password)

	// Forward request
	resp, err := p.httpClient.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("failed to forward request: %v", err)
	}
	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		w.Header()[name] = values
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body: %v", err)
	}

	return nil
}

// ErrorResponse represents a Docker Registry API error response
type ErrorResponse struct {
	Errors []ErrorDetail `json:"errors"`
}

// ErrorDetail represents a single error in the response
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// writeErrorResponse writes a Docker Registry API compliant error response
func writeErrorResponse(w http.ResponseWriter, code string, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Errors: []ErrorDetail{
			{
				Code:    code,
				Message: message,
			},
		},
	}

	json.NewEncoder(w).Encode(errorResp)
}