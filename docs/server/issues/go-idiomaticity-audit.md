# Go Idiomaticity Audit

## Executive Summary

After reviewing key files across the codebase, I've identified several patterns that deviate from idiomatic Go. While the code is functional, adopting Go conventions will improve maintainability, readability, and performance.

## High-Level Findings

### 1. Error Handling Patterns

**Non-idiomatic patterns found:**
- Wrapping errors with `fmt.Errorf` everywhere, even when context isn't added
- Inconsistent error message formatting (some capitalized, some not)
- Custom error types not implementing the `error` interface idiomatically

**Examples:**
```go
// Found in service.go
return nil, fmt.Errorf("validation failed: %w", err)
return nil, fmt.Errorf("failed to create user: %w", err)

// More idiomatic:
return nil, fmt.Errorf("create user: %w", err)
```

**Recommendation:** Follow the Go error handling conventions:
- Error messages should be lowercase and not end with punctuation
- Use concise prefixes that describe the operation
- Consider using sentinel errors for common cases

### 2. Interface Design

**Non-idiomatic patterns:**
- Large interfaces (Storage interface has 20+ methods)
- Not following the "accept interfaces, return structs" principle
- Missing smaller, composable interfaces

**Recommendation:** Break down large interfaces into smaller, focused ones:
```go
// Instead of one large Storage interface, consider:
type UserStore interface {
    CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
    GetUser(ctx context.Context, userID string) (*User, error)
}

type MemoryStore interface {
    CreateMemory(ctx context.Context, req CreateMemoryRequest) (*Memory, error)
    GetMemory(ctx context.Context, userID string, vaultID uuid.UUID, memoryID string) (*Memory, error)
}
```

### 3. Struct Field Tags and JSON Handling

**Non-idiomatic patterns:**
- Inconsistent use of `omitempty` in JSON tags
- Manual JSON marshaling/unmarshaling in places where struct tags would suffice
- Using `spanner.NullJSON` wrapper types extensively

**Example from http/memory.go:**
```go
// Current:
var req struct {
    Email       string  `json:"email"`
    DisplayName *string `json:"displayName,omitempty"`
    TimeZone    string  `json:"timeZone"`
}

// More idiomatic: Define proper request/response types
type CreateUserRequest struct {
    Email       string  `json:"email" validate:"required,email"`
    DisplayName *string `json:"displayName,omitempty"`
    TimeZone    string  `json:"timeZone,omitempty"`
}
```

### 4. Package Organization

**Non-idiomatic patterns:**
- Deep nesting of packages (internal/http)
- Mixing concerns within packages
- Not following the "internal" package convention properly

**Recommendation:** Flatten the package structure:
```
internal/
├── storage/      # All storage implementations
├── service/      # Business logic
├── http/         # HTTP handlers
├── config/       # Configuration
└── search/       # Search functionality
```

### 5. Testing Patterns

**Non-idiomatic patterns:**
- Test names don't follow Go conventions
- Not using table-driven tests effectively
- Missing test helpers and fixtures

**Example improvement:**
```go
// Current:
func TestCreateUser_InvalidEmail(t *testing.T) {
    if err := CreateUser("bad email", nil); err == nil {
        t.Fatalf("expected error for invalid email")
    }
}

// More idiomatic:
func TestCreateUser(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"valid email", "user@example.com", false},
        {"invalid email", "bad email", true},
        {"empty email", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := CreateUser(tt.email, nil)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 6. Concurrency and Context Usage

**Good patterns observed:**
- Proper context propagation through all layers
- No goroutine leaks found

**Areas for improvement:**
- Not using context for timeout/cancellation in some places
- Could benefit from context values for request-scoped data

### 7. Configuration Management

**Non-idiomatic patterns:**
- Using envconfig library instead of standard library
- Complex configuration resolution logic
- Not following 12-factor app principles fully

**Recommendation:** Simplify configuration:
```go
// Use standard library where possible
func LoadConfig() (*Config, error) {
    cfg := &Config{
        Port:     getEnvOrDefault("PORT", "8080"),
        Database: getEnvOrDefault("DATABASE_URL", ""),
    }
    return cfg, cfg.Validate()
}
```

### 8. HTTP Handler Patterns

**Non-idiomatic patterns:**
- Not using http.Handler interface effectively
- Manual JSON encoding/decoding instead of helper functions
- Inconsistent error response formats

**Recommendation:** Use middleware and helpers:
```go
// Create reusable middleware
func JSONMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        next.ServeHTTP(w, r)
    })
}
```

### 9. Database/Storage Patterns

**Non-idiomatic patterns:**
- Not using database/sql patterns
- Complex transaction handling
- Missing connection pooling configuration

### 10. Logging

**Good:** Using zerolog consistently

**Improvement:** Consider structured logging fields more consistently:
```go
// Instead of:
log.Info().Str("userID", userID).Msg("Creating memory")

// Consider:
logger := log.With().Str("userID", userID).Logger()
logger.Info().Msg("creating memory")
```

## Recommended Reading

As you're learning Go, I recommend these resources in order:

### 1. **Essential Reading**
- **"Effective Go"** - Official Go documentation (https://go.dev/doc/effective_go)
- **"The Go Programming Language"** by Donovan & Kernighan - The definitive book
- **"Go Code Review Comments"** - Common Go style issues (https://github.com/golang/go/wiki/CodeReviewComments)

### 2. **Error Handling**
- **"Working with Errors in Go 1.13"** - Blog post by the Go team
- **Dave Cheney's "Don't just check errors, handle them gracefully"**

### 3. **Design Patterns**
- **"Go Patterns"** - https://github.com/tmrts/go-patterns
- **"Uber Go Style Guide"** - Comprehensive style guide used at Uber

### 4. **Testing**
- **"Advanced Testing with Go"** by Mitchell Hashimoto
- **"Learn Go with Tests"** - https://quii.gitbook.io/learn-go-with-tests

### 5. **Concurrency**
- **"Concurrency in Go"** by Katherine Cox-Buday
- **Go Blog: "Go Concurrency Patterns"**

### 6. **API Design**
- **"Style guideline for Go packages"** - https://rakyll.org/style-packages/
- **"Standard Package Layout"** - https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1

## Priority Actions

1. **Immediate:** Fix error messages to be lowercase and concise
2. **Short-term:** Refactor large interfaces into smaller ones
3. **Medium-term:** Adopt table-driven tests throughout
4. **Long-term:** Restructure packages to be flatter and more focused

## Positive Observations

- Good use of context throughout
- Consistent code formatting
- Clear separation of concerns between layers
- Good documentation in critical sections
- Proper use of defer for cleanup

The codebase shows good software engineering practices overall. These improvements will make it more "Go-like" and easier for Go developers to understand and maintain.
