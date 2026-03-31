package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommercc/internal/product"
)

func TestBuildRouterKeepsRootPublicAndProductsProtected(t *testing.T) {
	productHandler := product.NewHandler(product.NewService())

	adminAuthCalled := false
	subadminAuthCalled := false
	router := buildRouter(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			adminAuthCalled = true
			w.WriteHeader(http.StatusUnauthorized)
		})
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			subadminAuthCalled = true
			w.WriteHeader(http.StatusForbidden)
		})
	}, productHandler)

	rootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rootRec := httptest.NewRecorder()
	router.ServeHTTP(rootRec, rootReq)

	if rootRec.Code != http.StatusOK {
		t.Fatalf("root returned %d, want %d", rootRec.Code, http.StatusOK)
	}

	productsReq := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	productsRec := httptest.NewRecorder()
	router.ServeHTTP(productsRec, productsReq)

	if productsRec.Code != http.StatusUnauthorized {
		t.Fatalf("products returned %d, want %d", productsRec.Code, http.StatusUnauthorized)
	}

	subadminReq := httptest.NewRequest(http.MethodGet, "/api/subadmin/products", nil)
	subadminRec := httptest.NewRecorder()
	router.ServeHTTP(subadminRec, subadminReq)

	if subadminRec.Code != http.StatusForbidden {
		t.Fatalf("subadmin products returned %d, want %d", subadminRec.Code, http.StatusForbidden)
	}

	if !adminAuthCalled {
		t.Fatal("expected admin auth middleware to be used for /api/products")
	}

	if !subadminAuthCalled {
		t.Fatal("expected subadmin auth middleware to be used for /api/subadmin/products")
	}
}
