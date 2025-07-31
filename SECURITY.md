# Security Policy

## ⚠️ Security Status

This is an **MVP implementation** for demonstration purposes. This codebase is NOT production-ready and lacks several critical security features.

## Known Security Limitations

### Currently NOT Implemented:
1. **Authentication** - All API endpoints are publicly accessible
2. **Authorization** - No user access control or data isolation
3. **Rate Limiting** - No protection against DoS attacks
4. **Input Sanitization** - Basic validation only
5. **HTTPS Enforcement** - Must be configured externally

### Partially Implemented:
- Basic input validation for length and format
- SQL parameterization (needs audit)
- Error handling (may leak information)

## Security Roadmap

See [Security MVP Roadmap](issues/security-mvp-roadmap.md) for our plan to address these issues:
- **Phase 1**: GitHub Ready - Basic security hygiene
- **Phase 2**: Production Ready - Full security implementation

## Reporting Security Issues

As this is an MVP, we're aware of the major security gaps. However, if you discover additional vulnerabilities:

1. **Do NOT** use the public issue tracker
2. **Do NOT** deploy this code with real user data
3. Create a private security advisory or contact the maintainers directly

## Deployment Warning

**DO NOT DEPLOY TO PRODUCTION** without implementing:
- JWT or OAuth2 authentication
- User-based authorization
- HTTPS/TLS encryption
- Rate limiting
- Comprehensive input validation

## For Developers

If you're contributing to this project:
1. Don't add hardcoded secrets
2. Use environment variables for configuration
3. Follow the validation patterns in `internal/api/validate`
4. Add tests for any security-related code

## Timeline

- **Current**: MVP for demonstration only
- **Before GitHub Public**: Remove hardcoded values, add disclaimers
- **Before Beta Users**: Implement authentication and basic security
- **Before GA**: Full security audit and penetration testing

Remember: This is a learning/demonstration project. Security will be properly implemented before any production use.
