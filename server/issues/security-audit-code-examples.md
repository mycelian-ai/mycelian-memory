# Security Audit - Vulnerable Code Examples

## 1. Authentication Bypass Example

**Vulnerable Code** (current implementation):
```go
// Anyone can access any user's data
func (h *MemoryHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    userID := vars["userId"]  // No verification this is the authenticated user!
    
    user, err := h.memoryService.GetUser(r.Context(), userID)
    // Returns any user's data without authentication
}
```

**Secure Implementation**:
```go
func (h *MemoryHandler) GetUser(w http.ResponseWriter, r *http.Request) {
    // Extract authenticated user from context (set by auth middleware)
    authUser, ok := r.Context().Value("user").(AuthenticatedUser)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    
    vars := mux.Vars(r)
    requestedUserID := vars["userId"]
    
    // Verify user can only access their own data
    if authUser.UserID != requestedUserID {
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }
    
    user, err := h.memoryService.GetUser(r.Context(), requestedUserID)
    // ... rest of implementation
}
```

## 2. SQL Injection Vulnerability

**Vulnerable Pattern** (found in storage layer):
```go
// DANGEROUS: String concatenation in SQL
query := `SELECT * FROM MemoryEntries WHERE UserId = @userId`
if req.Before != nil {
    query += " AND CreationTime < @before"  // String concatenation
}
if req.After != nil {
    query += " AND CreationTime > @after"   // More concatenation
}
```

**Secure Implementation**:
```go
// Use parameterized queries with proper builders
stmt := spanner.Statement{
    SQL: `SELECT * FROM MemoryEntries 
          WHERE UserId = @userId 
          AND (@before IS NULL OR CreationTime < @before)
          AND (@after IS NULL OR CreationTime > @after)`,
    Params: map[string]interface{}{
        "userId": req.UserID,
        "before": req.Before,  // Can be nil
        "after":  req.After,   // Can be nil
    },
}
```

## 3. Input Validation Issues

**Vulnerable Code**:
```go
// Weak email validation
var emailRx = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// No HTML/JS sanitization
type CreateMemoryEntryRequest struct {
    RawEntry string  // Could contain <script>alert('XSS')</script>
    Summary  string  // No sanitization
    Metadata map[string]interface{}  // Arbitrary JSON accepted
}
```

**Secure Implementation**:
```go
import (
    "github.com/microcosm-cc/bluemonday"
    "github.com/badoux/checkmail"
)

// Better email validation
func ValidateEmail(email string) error {
    if err := checkmail.ValidateFormat(email); err != nil {
        return fmt.Errorf("invalid email format: %w", err)
    }
    // Additional checks for disposable emails, etc.
    return nil
}

// HTML sanitization
func SanitizeUserInput(input string) string {
    p := bluemonday.StrictPolicy()
    return p.Sanitize(input)
}

// Validate metadata structure
func ValidateMetadata(metadata map[string]interface{}) error {
    // Define allowed keys
    allowedKeys := map[string]bool{
        "source": true,
        "tags": true,
        "version": true,
    }
    
    for key, value := range metadata {
        if !allowedKeys[key] {
            return fmt.Errorf("unexpected metadata key: %s", key)
        }
        // Type validation based on key
        switch key {
        case "tags":
            if _, ok := value.([]string); !ok {
                return fmt.Errorf("tags must be string array")
            }
        }
    }
    return nil
}
```

## 4. Missing Rate Limiting

**Current Code** (no rate limiting):
```go
router.HandleFunc("/api/users", memoryHandler.CreateUser).Methods("POST")
// Anyone can spam user creation
```

**Secure Implementation**:
```go
import "golang.org/x/time/rate"

// Rate limiter middleware
func RateLimitMiddleware(next http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 req/sec
    
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// Apply to routes
router.Use(RateLimitMiddleware)
```

## 5. Sensitive Data in Logs

**Vulnerable Code**:
```go
log.Info().
    Str("email", req.Email).        // PII in logs!
    Str("userID", req.UserID).      // Sensitive ID
    Msg("Creating user")
```

**Secure Implementation**:
```go
// Redact sensitive information
func redactEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return "***"
    }
    if len(parts[0]) > 2 {
        return parts[0][:2] + "***@" + parts[1]
    }
    return "***@" + parts[1]
}

log.Info().
    Str("email", redactEmail(req.Email)).  // Redacted
    Str("userID", hashUserID(req.UserID)). // Hashed
    Msg("Creating user")
```

## 6. Missing CSRF Protection

**Vulnerable Form**:
```html
<form action="/api/users/123/memories" method="POST">
    <input name="title" />
    <button type="submit">Create Memory</button>
</form>
```

**Secure Implementation**:
```go
// CSRF middleware
func CSRFMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" {
            token := r.Header.Get("X-CSRF-Token")
            if !validateCSRFToken(token, r) {
                http.Error(w, "Invalid CSRF token", http.StatusForbidden)
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}
```

## 7. Error Information Disclosure

**Vulnerable Code**:
```go
if err != nil {
    platformHttp.WriteInternalError(w, err.Error()) // Exposes internal error!
}
```

**Secure Implementation**:
```go
if err != nil {
    // Log detailed error internally
    log.Error().Err(err).Msg("Database operation failed")
    
    // Return generic error to client
    platformHttp.WriteInternalError(w, "An error occurred processing your request")
}
```

## 8. Missing Security Headers

**Add Security Headers Middleware**:
```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        
        next.ServeHTTP(w, r)
    })
}
```

## 9. Request Size Limits

**Secure Implementation**:
```go
func RequestSizeLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            
            // Also limit JSON depth
            decoder := json.NewDecoder(r.Body)
            decoder.DisallowUnknownFields()
            
            next.ServeHTTP(w, r)
        })
    }
}

// Apply 1MB limit
router.Use(RequestSizeLimitMiddleware(1 << 20))
```

## 10. Secure Docker Configuration

**Current Dockerfile Issues**:
```dockerfile
# Running as root (default)
FROM golang:1.24
# No security scanning
# No user creation
```

**Secure Dockerfile**:
```dockerfile
FROM golang:1.24-alpine AS builder

# Create non-root user
RUN adduser -D -g '' appuser

# Build application
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -o app

# Final stage
FROM scratch
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /build/app /app

# Use non-root user
USER appuser

# Security labels
LABEL security.scan="true"
LABEL security.nonroot="true"

ENTRYPOINT ["/app"]
```

## Complete Authentication Middleware Example

```go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/golang-jwt/jwt/v4"
)

type contextKey string

const userContextKey contextKey = "user"

type AuthClaims struct {
    UserID string `json:"user_id"`
    Email  string `json:"email"`
    jwt.RegisteredClaims
}

func AuthMiddleware(jwtSecret []byte) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract token from Authorization header
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Missing authorization header", http.StatusUnauthorized)
                return
            }
            
            // Verify Bearer prefix
            parts := strings.Split(authHeader, " ")
            if len(parts) != 2 || parts[0] != "Bearer" {
                http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
                return
            }
            
            tokenString := parts[1]
            
            // Parse and validate token
            token, err := jwt.ParseWithClaims(tokenString, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
                }
                return jwtSecret, nil
            })
            
            if err != nil || !token.Valid {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            // Extract claims
            claims, ok := token.Claims.(*AuthClaims)
            if !ok {
                http.Error(w, "Invalid token claims", http.StatusUnauthorized)
                return
            }
            
            // Add user info to context
            ctx := context.WithValue(r.Context(), userContextKey, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Helper to extract user from context
func GetUser(ctx context.Context) (*AuthClaims, bool) {
    user, ok := ctx.Value(userContextKey).(*AuthClaims)
    return user, ok
}
```

These examples demonstrate the security vulnerabilities found and provide secure implementation patterns that should be adopted throughout the codebase.
