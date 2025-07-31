# Security Implementation Roadmap

**Created**: January 26, 2025  
**Priority**: 🔴 CRITICAL - Block all other development until Phase 1 complete

## Executive Summary

This roadmap provides a step-by-step implementation plan to address the critical security vulnerabilities identified in the security audit. The most critical finding is the **complete absence of authentication**, which makes the system completely open to unauthorized access.

## Implementation Phases

### 🚨 Phase 0: Emergency Measures (1-2 days)

**Goal**: Prevent deployment to any public-facing environment

1. **Add deployment blockers**
   ```yaml
   # .github/workflows/deploy.yml
   - name: Security Check
     run: |
       echo "DEPLOYMENT BLOCKED: Critical security vulnerabilities"
       echo "See issues/security-audit-findings.md"
       exit 1
   ```

2. **Document security status**
   - Add WARNING to README.md
   - Update all deployment documentation
   - Notify all team members

3. **Inventory sensitive data**
   - Identify all PII stored
   - Document data flows
   - Assess breach impact

### 🔴 Phase 1: Authentication & Authorization (1 week)

**Goal**: Implement basic access control

#### 1.1 JWT Authentication (Days 1-3)

```go
// internal/auth/jwt.go
package auth

import (
    "github.com/golang-jwt/jwt/v4"
)

type JWTManager struct {
    secretKey []byte
    issuer    string
    audience  string
}

func (m *JWTManager) GenerateToken(userID, email string) (string, error) {
    claims := &AuthClaims{
        UserID: userID,
        Email:  email,
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    m.issuer,
            Subject:   userID,
            Audience:  []string{m.audience},
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secretKey)
}
```

#### 1.2 Auth Middleware (Days 2-3)

```go
// internal/api/middleware/auth.go
func AuthMiddleware(jwtManager *auth.JWTManager) mux.MiddlewareFunc {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip auth for health checks and login
            if r.URL.Path == "/api/health" || r.URL.Path == "/api/auth/login" {
                next.ServeHTTP(w, r)
                return
            }
            
            token := extractToken(r)
            if token == "" {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            
            claims, err := jwtManager.ValidateToken(token)
            if err != nil {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }
            
            ctx := context.WithValue(r.Context(), "user", claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

#### 1.3 Authorization Layer (Days 3-4)

```go
// internal/api/middleware/authz.go
func RequireOwnership(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := GetAuthUser(r.Context())
        vars := mux.Vars(r)
        requestedUserID := vars["userId"]
        
        if user.UserID != requestedUserID {
            http.Error(w, "Forbidden", http.StatusForbidden)
            return
        }
        
        next.ServeHTTP(w, r)
    }
}
```

#### 1.4 Login Endpoint (Days 4-5)

```go
// internal/api/http/auth.go
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Validate credentials (implement your auth strategy)
    user, err := h.authService.ValidateCredentials(req.Email, req.Password)
    if err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }
    
    // Generate token
    token, err := h.jwtManager.GenerateToken(user.UserID, user.Email)
    if err != nil {
        http.Error(w, "Token generation failed", http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(LoginResponse{
        Token: token,
        User:  user,
    })
}
```

#### 1.5 Update All Handlers (Days 5-7)

- Add ownership checks to all endpoints
- Validate user access to requested resources
- Remove userID from URL paths where possible

### 🟠 Phase 2: Input Validation & SQL Security (1 week)

#### 2.1 Comprehensive Input Validation (Days 1-3)

```go
// internal/api/validate/enhanced.go
package validate

import (
    "github.com/go-playground/validator/v10"
    "github.com/microcosm-cc/bluemonday"
)

var (
    validate = validator.New()
    policy   = bluemonday.StrictPolicy()
)

type CreateMemoryRequest struct {
    Title       string `json:"title" validate:"required,min=1,max=50,alphanumeric"`
    Description string `json:"description" validate:"max=2048"`
    MemoryType  string `json:"memoryType" validate:"required,oneof=personal work shared"`
}

func ValidateAndSanitize(req interface{}) error {
    if err := validate.Struct(req); err != nil {
        return err
    }
    
    // Sanitize string fields
    sanitizeStrings(req)
    return nil
}
```

#### 2.2 Fix SQL Injection Vulnerabilities (Days 3-5)

```go
// Replace all dynamic SQL with parameterized queries
// BAD:
query += " AND CreationTime < @before"

// GOOD:
stmt := spanner.Statement{
    SQL: `SELECT * FROM MemoryEntries 
          WHERE UserId = @userId 
          AND (@beforeTime IS NULL OR CreationTime < @beforeTime)`,
    Params: map[string]interface{}{
        "userId":     userID,
        "beforeTime": req.Before,
    },
}
```

#### 2.3 JSON Schema Validation (Days 5-7)

```go
// internal/api/validate/json.go
import "github.com/xeipuuv/gojsonschema"

var contextSchema = `{
    "type": "object",
    "properties": {
        "activeContext": {"type": "string", "minLength": 1},
        "summary": {"type": "string", "maxLength": 1000}
    },
    "required": ["activeContext"],
    "additionalProperties": false
}`

func ValidateJSON(data []byte, schema string) error {
    schemaLoader := gojsonschema.NewStringLoader(schema)
    documentLoader := gojsonschema.NewBytesLoader(data)
    
    result, err := gojsonschema.Validate(schemaLoader, documentLoader)
    if err != nil {
        return err
    }
    
    if !result.Valid() {
        return fmt.Errorf("JSON validation failed: %v", result.Errors())
    }
    
    return nil
}
```

### 🟡 Phase 3: Security Infrastructure (2 weeks)

#### 3.1 Rate Limiting (Days 1-3)

```go
// internal/api/middleware/ratelimit.go
import "github.com/ulule/limiter/v3"

func RateLimitMiddleware(store limiter.Store) mux.MiddlewareFunc {
    rate := limiter.Rate{
        Period: 1 * time.Minute,
        Limit:  60,
    }
    
    instance := limiter.New(store, rate)
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            context, err := instance.Get(r.Context(), getUserKey(r))
            if err != nil {
                http.Error(w, "Rate limit error", http.StatusInternalServerError)
                return
            }
            
            if context.Reached {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

#### 3.2 Security Headers (Days 3-4)

```go
// internal/api/middleware/security.go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        
        next.ServeHTTP(w, r)
    })
}
```

#### 3.3 Audit Logging (Days 5-7)

```go
// internal/audit/logger.go
type AuditLogger struct {
    storage AuditStorage
}

func (a *AuditLogger) LogRequest(r *http.Request, userID string, action string) {
    entry := AuditEntry{
        Timestamp: time.Now(),
        UserID:    userID,
        Action:    action,
        Resource:  r.URL.Path,
        Method:    r.Method,
        IP:        getClientIP(r),
        UserAgent: r.UserAgent(),
    }
    
    a.storage.Store(entry)
}
```

#### 3.4 Secrets Management (Days 8-10)

```go
// internal/config/secrets.go
import "github.com/hashicorp/vault/api"

type SecretsManager struct {
    client *api.Client
}

func (s *SecretsManager) GetJWTSecret() ([]byte, error) {
    secret, err := s.client.Logical().Read("secret/data/jwt")
    if err != nil {
        return nil, err
    }
    
    return []byte(secret.Data["key"].(string)), nil
}
```

#### 3.5 CSRF Protection (Days 11-14)

```go
// internal/api/middleware/csrf.go
import "github.com/gorilla/csrf"

func CSRFMiddleware(authKey []byte) mux.MiddlewareFunc {
    return csrf.Protect(
        authKey,
        csrf.Secure(true),
        csrf.HttpOnly(true),
        csrf.SameSite(csrf.SameSiteStrictMode),
    )
}
```

### 🟢 Phase 4: Testing & Monitoring (1 week)

#### 4.1 Security Test Suite

```go
// internal/api/security_test.go
func TestAuthenticationRequired(t *testing.T) {
    endpoints := []string{
        "/api/users/123",
        "/api/users/123/vaults",
        "/api/users/123/memories",
    }
    
    for _, endpoint := range endpoints {
        t.Run(endpoint, func(t *testing.T) {
            req := httptest.NewRequest("GET", endpoint, nil)
            // No auth header
            
            rr := httptest.NewRecorder()
            router.ServeHTTP(rr, req)
            
            assert.Equal(t, http.StatusUnauthorized, rr.Code)
        })
    }
}
```

#### 4.2 Penetration Testing Checklist

- [ ] Authentication bypass attempts
- [ ] SQL injection testing
- [ ] XSS payload testing
- [ ] CSRF attack simulation
- [ ] Rate limit testing
- [ ] Authorization boundary testing

#### 4.3 Security Monitoring

```yaml
# monitoring/alerts.yml
alerts:
  - name: HighFailedLoginRate
    expr: rate(login_failures[5m]) > 10
    severity: warning
    
  - name: UnauthorizedAccessAttempts
    expr: rate(unauthorized_requests[5m]) > 50
    severity: critical
    
  - name: SQLInjectionAttempt
    expr: sql_injection_detected > 0
    severity: critical
```

## Implementation Checklist

### Week 1: Authentication
- [ ] Implement JWT generation and validation
- [ ] Add authentication middleware
- [ ] Create login endpoint
- [ ] Add authorization checks to all endpoints
- [ ] Update API documentation

### Week 2: Input Security
- [ ] Implement comprehensive input validation
- [ ] Fix all SQL injection vulnerabilities
- [ ] Add JSON schema validation
- [ ] Implement HTML/JS sanitization
- [ ] Add request size limits

### Week 3-4: Infrastructure
- [ ] Implement rate limiting
- [ ] Add security headers
- [ ] Set up audit logging
- [ ] Implement secrets management
- [ ] Add CSRF protection

### Week 5: Testing & Deployment
- [ ] Complete security test suite
- [ ] Conduct penetration testing
- [ ] Set up security monitoring
- [ ] Update deployment pipeline
- [ ] Security training for team

## Success Criteria

1. **No unauthenticated access** to any user data
2. **All inputs validated** and sanitized
3. **No SQL injection** vulnerabilities
4. **Rate limiting** on all endpoints
5. **Security headers** on all responses
6. **Audit logging** for all actions
7. **Automated security tests** passing
8. **Security monitoring** in place

## Ongoing Security Practices

1. **Weekly security reviews**
2. **Automated dependency scanning**
3. **Regular penetration testing**
4. **Security training quarterly**
5. **Incident response drills**

## Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- [SQL Injection Prevention](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)

## Emergency Contacts

- Security Team Lead: [Contact]
- Incident Response: [Contact]
- Legal/Compliance: [Contact]

Remember: **Security is not optional**. Every day without authentication is a day of complete vulnerability.
