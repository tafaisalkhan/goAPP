package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ecommercc/internal/auth"
	"ecommercc/internal/product"
)

const serverAddr = ":8081"

func main() {
	loadEnvFile(".env")

	productService := product.NewService()
	productHandler := product.NewHandler(productService)

	keycloakAuth, err := auth.NewKeycloakAuth(auth.KeycloakConfig{
		Issuer:   os.Getenv("KEYCLOAK_ISSUER"),
		JWKSURL:  os.Getenv("KEYCLOAK_JWKS_URL"),
		Audience: os.Getenv("KEYCLOAK_AUDIENCE"),
	})
	if err != nil {
		log.Fatalf("keycloak auth: %v", err)
	}

	r := buildRouter(
		keycloakAuth.MiddlewareForRole("cloud-admin"),
		keycloakAuth.MiddlewareForRole("subadmin"),
		productHandler,
	)

	server := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}

	log.Println("server stopped")
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		_ = os.Setenv(key, value)
	}
}
