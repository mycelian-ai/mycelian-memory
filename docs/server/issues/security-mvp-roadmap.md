# Security Roadmap: MVP vs Production

**Created**: January 26, 2025  
**Context**: YC MVP Application - No real users yet

## Overview

This roadmap separates security requirements into two phases:
1. **GitHub Ready** - Minimum security for open-source visibility
2. **Production Ready** - Full security for real user data

## Phase 1: GitHub Ready (Before Making Repo Public)

### 🟡 Priority: Medium - Protect Your Reputation

#### 1.1 Code Hygiene (1-2 days)

**Remove Sensitive Data**:
```bash
# Check for hardcoded secrets
grep -r "password\|secret\|key\|token" --include="*.go" .

# Remove hardcoded values
# Current issue: internal/config/config.go has hardcoded GCP project
GCPProjectID string `envconfig:"GCP_PROJECT_ID" default:"artful-guru-459003-k4"`
```

**Add .gitignore entries**:
```gitignore
# Secrets
.env
*.key
*.pem
secrets/

# Local data
*.db
memory.db
.synapse-memory/

# Config overrides
config.local.yml
```

#### 1.2 Documentation (1 day)

**Add Security Disclaimer**:
```markdown
# README.md

## ⚠️ Security Notice

This is an MVP implementation for demonstration purposes. 
**DO NOT use this in production without implementing proper security measures.**

Key security features NOT yet implemented:
- Authentication & Authorization
- Input sanitization
- Rate limiting
- HTTPS enforcement

See [Security Roadmap](issues/security-mvp-roadmap.md) for planned improvements.
```

**Document Known Issues**:
```markdown
# SECURITY.md

## Known Security Issues

This MVP currently lacks:
1. **Authentication**: All endpoints are public
2. **Authorization**: No user access control
3. **Input Validation**: Basic validation only
4. **Rate Limiting**: No DoS protection

These will be addressed before production deployment.
```

#### 1.3 Basic Input Validation (2-3 days)

**Fix Critical Validation Gaps**:
```go
// Just prevent obvious attacks for demo
func (s *Service) validateCreateMemoryRequest(req CreateMemoryRequest) error {
    // Prevent script injection in demos
    if strings.Contains(req.Title, "<script>") {
        return fmt.Errorf("invalid characters in title")
    }
    
    // Basic length limits
    if len(req.Title) > 50 {
        return fmt.Errorf("title too long")
    }
    
    return nil
}
```

#### 1.4 Environment Configuration (1 day)

**Use Environment Variables**:
```go
// Remove all hardcoded values
type Config struct {
    GCPProjectID string `envconfig:"GCP_PROJECT_ID" required:"true"`
    // No defaults for sensitive values
}
```

**Add .env.example**:
```bash
# .env.example
MEMORY_SERVER_GCP_PROJECT_ID=your-project-id
MEMORY_SERVER_SPANNER_INSTANCE_ID=your-instance
MEMORY_SERVER_HTTP_PORT=8080
```

### GitHub Ready Checklist

- [ ] No hardcoded secrets or credentials
- [ ] Security disclaimer in README
- [ ] SECURITY.md with known issues
- [ ] Basic XSS prevention
- [ ] Environment-based configuration
- [ ] .env.example file
- [ ] Updated .gitignore

## Phase 2: Production Ready (Before Real Users)

### 🔴 Priority: Critical - Required for Production

#### 2.1 Authentication (1 week)

**Simple JWT Implementation**:
```go
// Start with basic JWT auth
// Can upgrade to OAuth2 later

// For MVP, even basic auth is better than none
type AuthConfig struct {
    JWTSecret string `envconfig:"JWT_SECRET" required:"true"`
    // Start simple, enhance later
}
```

**MVP Authentication Flow**:
1. Email/password login (can add OAuth later)
2. JWT token generation
3. Token validation middleware
4. Protect all user endpoints

#### 2.2 Authorization (3-4 days)

**Simple User-Based Access**:
```go
// MVP: Users can only access their own data
func RequireOwnership(userID string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authUser := GetAuthUser(r.Context())
            if authUser.ID != userID {
                http.Error(w, "Forbidden", 403)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

#### 2.3 Essential Security Headers (1 day)

```go
// Minimum viable security headers
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        // Add more as needed
        next.ServeHTTP(w, r)
    })
}
```

#### 2.4 Basic Rate Limiting (1-2 days)

```go
// Simple in-memory rate limiting for MVP
// Can upgrade to Redis-based later
var limiter = rate.NewLimiter(10, 100) // 10 req/sec, burst 100

func RateLimit(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", 429)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

#### 2.5 HTTPS Only (1 day)

```go
// Force HTTPS in production
if config.Environment == "production" {
    router.Use(enforceHTTPS)
}
```

### Production Ready Checklist

- [ ] JWT authentication implemented
- [ ] User can only access own data
- [ ] Security headers added
- [ ] Rate limiting enabled
- [ ] HTTPS enforced
- [ ] SQL injection fixes
- [ ] Input validation enhanced
- [ ] Error messages sanitized

## MVP Development Strategy

### What You Can Defer

These can wait until after YC demo/funding:

1. **OAuth2/SSO** - JWT is fine for MVP
2. **Advanced RBAC** - Simple ownership check suffices
3. **Audit Logging** - Nice to have, not critical
4. **CSRF Protection** - If API-only, less critical
5. **Penetration Testing** - Do after initial traction
6. **Compliance (GDPR, etc.)** - Address when you have EU users

### What You Cannot Defer

Must have before any real users:

1. **Authentication** - Non-negotiable
2. **Authorization** - Users must be isolated
3. **HTTPS** - Basic security requirement
4. **Input Validation** - Prevent obvious attacks
5. **Rate Limiting** - Prevent abuse

## Practical Timeline

### For YC Demo (Current State OK)

- Run locally or on private infrastructure
- Use test data only
- Focus on functionality over security

### For GitHub Release (1 week)

- Clean up code (Phase 1)
- Add disclaimers
- Remove sensitive data
- Basic validation

### For Beta Users (2-3 weeks)

- Implement authentication
- Add authorization
- Enable HTTPS
- Basic rate limiting

### For General Availability (1-2 months)

- Full security audit
- Penetration testing
- Compliance review
- Advanced features

## Quick Wins for Demo

If you need to show security awareness in YC demo:

```go
// 1. Add a simple API key check (temporary)
func APIKeyAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        key := r.Header.Get("X-API-Key")
        if key != os.Getenv("DEMO_API_KEY") {
            http.Error(w, "Invalid API Key", 401)
            return
        }
        next.ServeHTTP(w, r)
    })
}

// 2. Log security events (shows awareness)
log.Info().
    Str("action", "api_access").
    Str("ip", r.RemoteAddr).
    Msg("API access logged for security monitoring")

// 3. Add basic headers (easy win)
w.Header().Set("X-Content-Type-Options", "nosniff")
```

## Communication Strategy

### For YC Application

"We've architected the system with security in mind, following OWASP best practices. While our MVP focuses on core functionality, we have a clear security roadmap for production deployment including JWT authentication, rate limiting, and comprehensive input validation."

### For GitHub README

"This is an MVP implementation. See our [Security Roadmap](issues/security-mvp-roadmap.md) for planned security enhancements before production use."

### For Investors

"Security is built into our architecture. We're following a phased approach - MVP for validation, then implementing enterprise-grade security before scaling."

## Resources for Quick Implementation

- [JWT in 100 lines of Go](https://github.com/golang-jwt/jwt)
- [Simple Rate Limiter](https://github.com/ulule/limiter)
- [Basic Auth Middleware](https://github.com/gorilla/mux#middleware)
- [Security Headers](https://securityheaders.com/)

## Remember

1. **Perfect is the enemy of good** - Basic auth is better than no auth
2. **Document everything** - Shows you're aware and planning
3. **Fix before scaling** - OK to defer for demo, not for real users
4. **Security is a journey** - Start simple, improve iteratively

The goal is to demonstrate competence and planning, not to build Fort Knox for a demo.
