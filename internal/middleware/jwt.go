package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWKSKey represents a single key from a JWKS response.
type JWKSKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// JWKS is the JSON Web Key Set response.
type JWKS struct {
	Keys []JWKSKey `json:"keys"`
}

// Claims holds the JWT token claims.
type Claims struct {
	Sub      string   `json:"sub"`
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	Iss      string   `json:"iss"`
	jwt.RegisteredClaims
}

// JWTValidator validates JWT tokens using korauth's JWKS.
type JWTValidator struct {
	authURL string
}

// NewJWTValidator creates a new JWTValidator.
func NewJWTValidator(authURL string) *JWTValidator {
	return &JWTValidator{authURL: authURL}
}

// Middleware returns a Gin handler that validates JWT tokens.
// Public paths (login, refresh, jwks) are skipped.
func (v *JWTValidator) Middleware(publicPaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicPath(c.Request.URL.Path, publicPaths) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "UNAUTHORIZED",
					"message":    "Authorization header missing",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		const bearerScheme = "Bearer "
		if !strings.HasPrefix(authHeader, bearerScheme) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "INVALID_TOKEN",
					"message":    "Invalid authorization scheme",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		tokenStr := authHeader[len(bearerScheme):]

		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			return v.getPublicKey(token.Header["kid"].(string))
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "INVALID_TOKEN",
					"message":    "Token validation failed",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok || claims.Iss != "openkor-auth" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":       "INVALID_TOKEN",
					"message":    "Invalid token claims",
					"request_id": c.GetString("request_id"),
				},
			})
			c.Abort()
			return
		}

		// Store claims in context for downstream middleware and handlers
		c.Set("user_id", claims.Sub)
		c.Set("tenant_id", claims.TenantID)
		c.Set("roles", claims.Roles)

		// Forward identity to backend services via headers
		c.Header("X-User-ID", claims.Sub)
		c.Header("X-Tenant-ID", claims.TenantID)
		c.Header("X-Roles", strings.Join(claims.Roles, ","))

		c.Next()
	}
}

// getPublicKey fetches the RSA public key matching the given kid from korauth's JWKS.
// TODO: cache the JWKS response in Redis/in-memory with TTL.
func (v *JWTValidator) getPublicKey(kid string) (interface{}, error) {
	resp, err := http.Get(fmt.Sprintf("%s/v1/auth/.well-known/jwks.json", v.authURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch JWKS: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return nil, err
	}

	for _, key := range jwks.Keys {
		if key.Kid == kid {
			if key.Kty != "RSA" {
				return nil, fmt.Errorf("unsupported key type: %s", key.Kty)
			}
			return jwkToRSAPublicKey(key)
		}
	}

	return nil, fmt.Errorf("key not found: %s", kid)
}

// jwkToRSAPublicKey converts a JWK into an *rsa.PublicKey.
func jwkToRSAPublicKey(key JWKSKey) (*rsa.PublicKey, error) {
	nData, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	n := new(big.Int)
	n.SetBytes(nData)

	eData, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	e := 0
	for _, b := range eData {
		e = e*256 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

// isPublicPath reports whether the given path is in the public paths list.
func isPublicPath(path string, publicPaths []string) bool {
	for _, public := range publicPaths {
		if strings.HasPrefix(path, public) {
			return true
		}
	}
	return false
}

// LoggingMiddleware logs each request's method, path, status, and duration.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// TODO: replace with structured slog JSON output
		fmt.Printf("[%s] %s %s - %d (%dms)\n",
			c.GetString("request_id"),
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration.Milliseconds(),
		)
	}
}

// SecurityHeaders sets common security response headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Next()
	}
}
