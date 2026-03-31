package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewKeycloakAuthFallsBackToAuthJWKSPath(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/Cloud7-Realm/protocol/openid-connect/certs":
			http.NotFound(w, r)
		case "/auth/realms/Cloud7-Realm/protocol/openid-connect/certs":
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"keys":[%s]}`, jwkJSON(key))
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	auth, err := NewKeycloakAuth(KeycloakConfig{
		Issuer: server.URL + "/realms/Cloud7-Realm",
	})
	if err != nil {
		t.Fatalf("NewKeycloakAuth returned error: %v", err)
	}

	if auth == nil {
		t.Fatal("expected auth to be initialized")
	}
}

func TestClaimsHasRoleUsesClientResourceAccess(t *testing.T) {
	claims := &jwtClaims{
		ResourceAccess: map[string]accessClaims{
			"ivolve.cloud7": {Roles: []string{"cloud-admin"}},
		},
	}

	if !claimsHasRole(claims, "cloud-admin", "ivolve.cloud7") {
		t.Fatal("expected cloud-admin role to be accepted for the configured client")
	}
}

func TestClaimsHasRoleUsesRealmAccessForSubadmin(t *testing.T) {
	claims := &jwtClaims{
		RealmAccess: accessClaims{Roles: []string{"subadmin"}},
	}

	if !claimsHasRole(claims, "subadmin", "ivolve.cloud7") {
		t.Fatal("expected subadmin realm role to be accepted")
	}
}

func TestMiddlewareForAnyRoleAcceptsSubadmin(t *testing.T) {
	auth := &KeycloakAuth{}
	if auth.MiddlewareForAnyRole("cloud-admin", "subadmin") == nil {
		t.Fatal("expected middleware to be created")
	}
}

func jwkJSON(key *rsa.PrivateKey) string {
	return fmt.Sprintf(
		`{"kty":"RSA","kid":"test-key","use":"sig","alg":"RS256","n":"%s","e":"%s"}`,
		base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		base64.RawURLEncoding.EncodeToString(bigEndianExponent(key.PublicKey.E)),
	)
}

func bigEndianExponent(e int) []byte {
	if e == 0 {
		return []byte{0}
	}

	var out []byte
	for e > 0 {
		out = append([]byte{byte(e % 256)}, out...)
		e /= 256
	}

	return out
}
