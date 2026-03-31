package main

import (
	"net/http"

	"ecommercc/internal/country"
	"ecommercc/internal/product"
	"ecommercc/internal/userdetail"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type middlewareFunc func(http.Handler) http.Handler

func buildRouter(adminAuthMiddleware middlewareFunc, subadminAuthMiddleware middlewareFunc, userDetailAuthMiddleware middlewareFunc, productHandler *product.Handler, userDetailHandler *userdetail.Handler, countryHandler *country.Handler) http.Handler {
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
	registerUserDetailRoutes(r, userDetailAuthMiddleware, userDetailHandler, "/api/user-details")
	registerUserDetailRoutes(r, userDetailAuthMiddleware, userDetailHandler, "/api/users-detail")
	registerCountryRoutes(r, userDetailAuthMiddleware, countryHandler, "/api/countries")

	return r
}

func registerUserDetailRoutes(r chi.Router, auth middlewareFunc, handler *userdetail.Handler, basePath string) {
	r.With(auth).Get(basePath, handler.List)
	r.With(auth).Post(basePath, handler.Create)
	r.With(auth).Get(basePath+"/{id}", handler.Get)
	r.With(auth).Put(basePath+"/{id}", handler.Update)
	r.With(auth).Delete(basePath+"/{id}", handler.Delete)
}

func registerCountryRoutes(r chi.Router, auth middlewareFunc, handler *country.Handler, basePath string) {
	r.With(auth).Get(basePath, handler.List)
	r.With(auth).Post(basePath, handler.Create)
	r.With(auth).Get(basePath+"/{id}", handler.Get)
	r.With(auth).Put(basePath+"/{id}", handler.Update)
	r.With(auth).Delete(basePath+"/{id}", handler.Delete)
}
