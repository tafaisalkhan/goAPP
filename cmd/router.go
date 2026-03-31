package main

import (
	"net/http"

	"ecommercc/internal/product"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type middlewareFunc func(http.Handler) http.Handler

func buildRouter(adminAuthMiddleware middlewareFunc, subadminAuthMiddleware middlewareFunc, productHandler *product.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"webserver is running"}`))
	})

	r.With(adminAuthMiddleware).Get("/api/products", productHandler.List)
	r.With(subadminAuthMiddleware).Get("/api/subadmin/products", productHandler.List)

	return r
}
