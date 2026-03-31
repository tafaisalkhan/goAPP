package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type KeycloakConfig struct {
	Issuer   string
	JWKSURL  string
	Audience string
}

type KeycloakAuth struct {
	cfg  KeycloakConfig
	http *http.Client

	mu    sync.RWMutex
	keys  map[string]*rsa.PublicKey
	clock func() time.Time
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

type jwtClaims struct {
	Iss            string                  `json:"iss"`
	Sub            string                  `json:"sub"`
	Azp            string                  `json:"azp"`
	Aud            any                     `json:"aud"`
	Exp            int64                   `json:"exp"`
	Nbf            int64                   `json:"nbf"`
	Iat            int64                   `json:"iat"`
	RealmAccess    accessClaims            `json:"realm_access"`
	ResourceAccess map[string]accessClaims `json:"resource_access"`
}

type accessClaims struct {
	Roles []string `json:"roles"`
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewKeycloakAuth(cfg KeycloakConfig) (*KeycloakAuth, error) {
	if cfg.Issuer == "" {
		return nil, errors.New("keycloak issuer is required")
	}
	if cfg.JWKSURL == "" {
		cfg.JWKSURL = strings.TrimRight(cfg.Issuer, "/") + "/protocol/openid-connect/certs"
	}

	a := &KeycloakAuth{
		cfg:  cfg,
		http: &http.Client{Timeout: 5 * time.Second},
		keys: make(map[string]*rsa.PublicKey),
		clock: func() time.Time {
			return time.Now()
		},
	}

	if err := a.refreshKeys(); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *KeycloakAuth) Middleware(next http.Handler) http.Handler {
	return a.MiddlewareForRole("cloud-admin")(next)
}

func (a *KeycloakAuth) MiddlewareForRole(role string) func(http.Handler) http.Handler {
	return a.MiddlewareForAnyRole(role)
}

func (a *KeycloakAuth) MiddlewareForAnyRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := bearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, err.Error())
				return
			}

			claims, err := a.verify(token)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid access token: "+err.Error())
				return
			}

			if a.cfg.Audience != "" && !claimsMatchesAudience(claims, a.cfg.Audience) {
				writeAuthError(w, http.StatusForbidden, "token audience is not allowed")
				return
			}

			for _, role := range roles {
				if claimsHasRole(claims, role, a.cfg.Audience) {
					next.ServeHTTP(w, r)
					return
				}
			}

			if len(roles) == 1 {
				writeAuthError(w, http.StatusForbidden, roles[0]+" role is required")
				return
			}

			writeAuthError(w, http.StatusForbidden, "required role is missing")
		})
	}
}

func (a *KeycloakAuth) verify(token string) (*jwtClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, err
	}
	if header.Alg != "RS256" || header.Kid == "" {
		return nil, errors.New("unsupported token header")
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var claims jwtClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, err
	}
	if !issuerMatches(claims.Iss, a.cfg.Issuer) {
		return nil, errors.New("invalid issuer")
	}
	now := a.clock().Unix()
	if claims.Exp != 0 && now >= claims.Exp {
		return nil, errors.New("token expired")
	}
	if claims.Nbf != 0 && now < claims.Nbf {
		return nil, errors.New("token not valid yet")
	}

	pub, err := a.publicKey(header.Kid)
	if err != nil {
		return nil, err
	}

	signingInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], signature); err != nil {
		return nil, err
	}

	return &claims, nil
}

func (a *KeycloakAuth) publicKey(kid string) (*rsa.PublicKey, error) {
	a.mu.RLock()
	pub, ok := a.keys[kid]
	a.mu.RUnlock()
	if ok {
		return pub, nil
	}

	if err := a.refreshKeys(); err != nil {
		return nil, err
	}

	a.mu.RLock()
	pub, ok = a.keys[kid]
	a.mu.RUnlock()
	if !ok {
		return nil, errors.New("signing key not found")
	}

	return pub, nil
}

func (a *KeycloakAuth) refreshKeys() error {
	var lastErr error

	for _, jwksURL := range a.jwksCandidates() {
		keys, err := a.loadKeys(jwksURL)
		if err == nil {
			a.mu.Lock()
			a.keys = keys
			a.mu.Unlock()

			return nil
		}

		lastErr = err
		if !errors.Is(err, errJWKSNotFound) {
			return err
		}
	}

	if lastErr != nil {
		return lastErr
	}

	return errors.New("unable to load jwks keys")
}

var errJWKSNotFound = errors.New("jwks endpoint not found")

func (a *KeycloakAuth) jwksCandidates() []string {
	candidates := []string{a.cfg.JWKSURL}

	if strings.Contains(a.cfg.JWKSURL, "/realms/") && !strings.Contains(a.cfg.JWKSURL, "/auth/realms/") {
		candidates = append(candidates, strings.Replace(a.cfg.JWKSURL, "/realms/", "/auth/realms/", 1))
	}

	if strings.Contains(a.cfg.JWKSURL, "/auth/realms/") {
		candidates = append(candidates, strings.Replace(a.cfg.JWKSURL, "/auth/realms/", "/realms/", 1))
	}

	return uniqueStrings(candidates)
}

func (a *KeycloakAuth) loadKeys(jwksURL string) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequest(http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errJWKSNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks endpoint %s returned %s", jwksURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var set jwksResponse
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))
	for _, key := range set.Keys {
		if key.Kid == "" || key.Kty != "RSA" {
			continue
		}

		pub, err := jwkToRSAPublicKey(key)
		if err != nil {
			continue
		}
		keys[key.Kid] = pub
	}

	if len(keys) == 0 {
		return nil, errors.New("no valid signing keys found")
	}

	return keys, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func jwkToRSAPublicKey(key jwk) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, err
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, err
	}

	e := 0
	for _, b := range eBytes {
		e = e*256 + int(b)
	}
	if e == 0 {
		return nil, errors.New("invalid exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}

func bearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("authorization header is required")
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("authorization header must use Bearer scheme")
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", errors.New("bearer token is required")
	}

	return token, nil
}

func hasAudience(aud any, expected string) bool {
	switch v := aud.(type) {
	case string:
		return subtle.ConstantTimeCompare([]byte(v), []byte(expected)) == 1
	case []any:
		for _, item := range v {
			s, ok := item.(string)
			if ok && subtle.ConstantTimeCompare([]byte(s), []byte(expected)) == 1 {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func claimsMatchesAudience(claims *jwtClaims, expected string) bool {
	return hasAudience(claims.Aud, expected) || stringEquals(claims.Azp, expected)
}

func claimsHasRole(claims *jwtClaims, expectedRole, clientID string) bool {
	if roleInList(claims.RealmAccess.Roles, expectedRole) {
		return true
	}

	if clientID == "" {
		for _, access := range claims.ResourceAccess {
			if roleInList(access.Roles, expectedRole) {
				return true
			}
		}
		return false
	}

	access, ok := claims.ResourceAccess[clientID]
	if !ok {
		return false
	}

	return roleInList(access.Roles, expectedRole)
}

func issuerMatches(actual, expected string) bool {
	return canonicalIssuer(actual) == canonicalIssuer(expected)
}

func canonicalIssuer(value string) string {
	value = strings.TrimRight(value, "/")
	value = strings.Replace(value, "/auth/realms/", "/realms/", 1)
	return value
}

func stringEquals(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func roleInList(roles []string, expected string) bool {
	for _, role := range roles {
		if stringEquals(role, expected) {
			return true
		}
	}

	return false
}

func writeAuthError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	body, _ := json.Marshal(map[string]string{"error": message})
	_, _ = w.Write(body)
}
