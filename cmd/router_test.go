package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommercc/internal/product"
	"ecommercc/internal/country"
	"ecommercc/internal/userdetail"
)

func TestBuildRouterKeepsRootPublicAndProductsProtected(t *testing.T) {
	productHandler := product.NewHandler(product.NewService(nil))
	userDetailHandler := userdetail.NewHandler(nil)
	countryHandler := country.NewHandler(nil)

	adminAuthCalled := false
	subadminAuthCalled := false
	userDetailAuthCalled := false
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
	}, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userDetailAuthCalled = true
			w.WriteHeader(http.StatusUnauthorized)
		})
	}, productHandler, userDetailHandler, countryHandler)

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

	userDetailsReq := httptest.NewRequest(http.MethodGet, "/api/user-details", nil)
	userDetailsRec := httptest.NewRecorder()
	router.ServeHTTP(userDetailsRec, userDetailsReq)

	if userDetailsRec.Code != http.StatusUnauthorized {
		t.Fatalf("user-details returned %d, want %d", userDetailsRec.Code, http.StatusUnauthorized)
	}

	if !userDetailAuthCalled {
		t.Fatal("expected user detail auth middleware to be used for /api/user-details")
	}

	userDetailsAliasReq := httptest.NewRequest(http.MethodGet, "/api/users-detail", nil)
	userDetailsAliasRec := httptest.NewRecorder()
	router.ServeHTTP(userDetailsAliasRec, userDetailsAliasReq)

	if userDetailsAliasRec.Code != http.StatusUnauthorized {
		t.Fatalf("users-detail returned %d, want %d", userDetailsAliasRec.Code, http.StatusUnauthorized)
	}

	countryReq := httptest.NewRequest(http.MethodGet, "/api/countries", nil)
	countryRec := httptest.NewRecorder()
	router.ServeHTTP(countryRec, countryReq)

	if countryRec.Code != http.StatusUnauthorized {
		t.Fatalf("countries returned %d, want %d", countryRec.Code, http.StatusUnauthorized)
	}
}
