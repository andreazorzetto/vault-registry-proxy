package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"

	"vault-docker-proxy/pkg/auth"
	"vault-docker-proxy/pkg/registry"
	"vault-docker-proxy/pkg/vault"
)

const (
	DefaultPort      = "8080"
	DefaultVaultAddr = "http://localhost:8200"
	DefaultRealm     = "https://auth.docker.io/token"
	DefaultService   = "registry.docker.io"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		vaultAddr = DefaultVaultAddr
	}

	log.Printf("Starting vault-docker-proxy on port %s", port)
	log.Printf("Vault address: %s", vaultAddr)

	// Create Vault client
	vaultClient, err := vault.NewClient(vaultAddr)
	if err != nil {
		log.Fatalf("Failed to create Vault client: %v", err)
	}

	// Create proxy server
	proxyServer := registry.NewProxyServer(vaultClient)

	// Setup routes with middleware
	router := setupRoutes(proxyServer)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	log.Fatal(server.ListenAndServe())
}

func setupRoutes(proxyServer *registry.ProxyServer) *mux.Router {
	r := mux.NewRouter()

	// Create authentication middleware
	authMiddleware := auth.NewMiddleware(DefaultRealm, DefaultService)

	// Apply middleware to all routes
	r.Use(authMiddleware.DockerRegistryAuth)

	// Docker Registry v2 API endpoints
	r.HandleFunc("/v2/", proxyServer.APIVersionCheck).Methods("GET")
	r.HandleFunc("/v2/_catalog", proxyServer.GetCatalog).Methods("GET")
	r.HandleFunc("/v2/{name:.*}/tags/list", proxyServer.GetTags).Methods("GET")
	r.HandleFunc("/v2/{name:.*}/manifests/{reference}", proxyServer.GetManifest).Methods("GET")
	r.HandleFunc("/v2/{name:.*}/blobs/{digest}", proxyServer.GetBlob).Methods("GET")

	return r
}