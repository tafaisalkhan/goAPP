package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommercc/internal/auth"
	"ecommercc/internal/country"
	"ecommercc/internal/config"
	"ecommercc/internal/product"
	"ecommercc/internal/userdetail"

	_ "github.com/go-sql-driver/mysql"
)

const serverAddr = ":8081"

func main() {
	loadEnvFile(".env")
	loadEnvFile("db.properties")

	db, err := openDatabase()
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	if err := ensureProductSchema(db); err != nil {
		log.Fatalf("database schema: %v", err)
	}
	if err := userdetail.EnsureSchema(db); err != nil {
		log.Fatalf("database schema: %v", err)
	}
	if err := country.EnsureSchema(db); err != nil {
		log.Fatalf("database schema: %v", err)
	}

	productService := product.NewService(db)
	productHandler := product.NewHandler(productService)
	userDetailRepo := userdetail.NewRepository(db)
	userDetailHandler := userdetail.NewHandler(userDetailRepo)
	countryRepo := country.NewRepository(db)
	countryHandler := country.NewHandler(countryRepo)

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
		keycloakAuth.MiddlewareForAnyRole("cloud-admin", "subadmin"),
		productHandler,
		userDetailHandler,
		countryHandler,
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
	_ = config.LoadPropertiesFile(path)
}

func openDatabase() (*sql.DB, error) {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "3306"
	}

	name := os.Getenv("DB_NAME")
	if name == "" {
		name = "cloud7"
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		user = "root"
	}

	password := os.Getenv("DB_PASSWORD")

	dsn := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + name + "?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	log.Printf("database connected: host=%s port=%s name=%s user=%s", host, port, name, user)

	return db, nil
}

func ensureProductSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			price_cents BIGINT NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&count); err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	_, err = db.Exec(`
		INSERT INTO products (id, name, description, price_cents) VALUES
		(1, 'Sample Product', 'Static product record', 1999),
		(2, 'Second Product', 'Another static record', 2999),
		(3, 'Third Product', 'Third static record', 3999)
	`)
	return err
}
