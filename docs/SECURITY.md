# GOLLM Security Guide

## Table of Contents

- [Security Overview](#security-overview)
- [Security Architecture](#security-architecture)
- [Credential Management](#credential-management)
- [Network Security](#network-security)
- [Input Validation](#input-validation)
- [Rate Limiting & Circuit Breakers](#rate-limiting--circuit-breakers)
- [Audit Logging](#audit-logging)
- [Configuration Security](#configuration-security)
- [Environment Security](#environment-security)
- [Security Testing](#security-testing)
- [Compliance](#compliance)
- [Threat Model](#threat-model)
- [Incident Response](#incident-response)
- [Best Practices](#best-practices)

## Security Overview

GOLLM implements enterprise-grade security features to protect sensitive data, prevent unauthorized access, and ensure secure communication with LLM providers. This guide covers all security aspects and provides best practices for secure deployment.

### Security Principles

1. **Defense in Depth**: Multiple layers of security controls
2. **Principle of Least Privilege**: Minimal access rights by default
3. **Security by Design**: Security built into the architecture
4. **Zero Trust**: Never trust, always verify
5. **Fail Secure**: Secure defaults and safe failure modes

### Security Features

- ✅ Secure credential storage with memory clearing
- ✅ TLS 1.3 minimum with certificate validation
- ✅ Comprehensive input validation and sanitization
- ✅ Rate limiting with token bucket algorithm
- ✅ Circuit breaker pattern for resilience
- ✅ Secure audit logging (no credential exposure)
- ✅ Path traversal and injection attack prevention
- ✅ Secure configuration management
- ✅ Memory-safe credential handling

## Security Architecture

### Core Security Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Nexus-LLM Security Stack                     │
├─────────────────────────────────────────────────────────────┤
│ CLI Interface                                               │
│ ├─ Input Validation & Sanitization                         │
│ ├─ Command Authorization                                    │
│ └─ Output Security (credential masking)                    │
├─────────────────────────────────────────────────────────────┤
│ Configuration Layer                                         │
│ ├─ Secure Config Storage (SecureString)                    │
│ ├─ Environment Variable Validation                         │
│ └─ Configuration Encryption                                │
├─────────────────────────────────────────────────────────────┤
│ Provider Security                                           │
│ ├─ API Key Management & Rotation                          │
│ ├─ Rate Limiting (Token Bucket)                           │
│ ├─ Circuit Breaker Pattern                                │
│ └─ Request/Response Validation                             │
├─────────────────────────────────────────────────────────────┤
│ Network Security                                            │
│ ├─ TLS 1.3 Minimum Enforcement                            │
│ ├─ Certificate Pinning                                     │
│ ├─ Timeout Management                                      │
│ └─ Connection Pooling Security                             │
├─────────────────────────────────────────────────────────────┤
│ Audit & Monitoring                                          │
│ ├─ Secure Audit Logging                                   │
│ ├─ Security Event Detection                               │
│ ├─ Performance Monitoring                                 │
│ └─ Anomaly Detection                                       │
└─────────────────────────────────────────────────────────────┘
```

## Credential Management

### Secure API Key Storage

GOLLM uses the `SecureString` type to handle sensitive credentials:

```go
// SecureString provides secure storage for sensitive data
type SecureString struct {
    value []byte
    mutex sync.RWMutex
}

// Automatically cleared from memory when no longer needed
```

#### Best Practices

1. **Environment Variables (Recommended)**
   ```bash
   # Use environment variables for API keys
   export OPENAI_API_KEY="sk-your-secure-key"
   export ANTHROPIC_API_KEY="your-anthropic-key"
   
   # Avoid shell history exposure
   set +o history
   export OPENAI_API_KEY="sk-your-key"
   set -o history
   ```

2. **Secure Configuration Storage**
   ```bash
   # Store API keys securely (encrypted on disk)
   gollm config set --secure providers.openai.api_key "sk-your-key"
   
   # Verify secure storage
   gollm config get providers.openai.api_key  # Shows masked value
   ```

3. **Key Rotation**
   ```bash
   # Rotate API keys regularly
   gollm config set --secure providers.openai.api_key "sk-new-key"
   gollm config validate  # Verify new key works
   ```

#### Credential Security Features

- **Memory Clearing**: Automatic memory clearing of sensitive data
- **Masked Display**: API keys are never displayed in logs or output
- **Secure Storage**: Encrypted storage in configuration files
- **Environment Priority**: Environment variables take precedence
- **Validation**: Automatic validation of API key formats

### Configuration Security

#### File Permissions

```bash
# Secure configuration directory
chmod 700 ~/.gollm
chmod 600 ~/.gollm/config.yaml

# System-wide configuration
sudo chmod 644 /etc/gollm/config.yaml
sudo chown root:root /etc/gollm/config.yaml
```

#### Secure Configuration Example

```yaml
# ~/.gollm/config.yaml - Secure configuration
default_provider: openai

providers:
  openai:
    api_key: "${OPENAI_API_KEY}"  # Use environment variable
    base_url: "https://api.openai.com/v1"
    
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"

# Security settings
security:
  tls_min_version: "1.3"
  certificate_pinning: true
  request_timeout: "30s"
  max_retries: 3
  
# Rate limiting configuration  
rate_limiting:
  enabled: true
  requests_per_minute: 60
  burst_size: 10
  
# Audit logging
audit:
  enabled: true
  log_requests: true
  log_responses: false  # Don't log response content
  mask_credentials: true
```

## Network Security

### TLS Configuration

GOLLM enforces TLS 1.3 minimum for all network communications:

```go
// TLS Configuration
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS13,
    CurvePreferences: []tls.CurveID{
        tls.X25519,
        tls.CurveP256,
    },
    CipherSuites: []uint16{
        tls.TLS_AES_256_GCM_SHA384,
        tls.TLS_CHACHA20_POLY1305_SHA256,
        tls.TLS_AES_128_GCM_SHA256,
    },
}
```

#### Certificate Validation

```yaml
# Enable certificate pinning for critical providers
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    tls:
      verify_certificates: true
      pin_certificates: true
      ca_bundle: "/path/to/ca-bundle.crt"  # Optional custom CA
```

#### Network Timeouts

```yaml
# Configure secure timeouts
settings:
  connect_timeout: "10s"
  request_timeout: "300s"  
  idle_timeout: "90s"
  
# Per-provider timeouts
providers:
  openai:
    timeout: "120s"
    dial_timeout: "10s"
    tls_handshake_timeout: "10s"
```

## Input Validation

GOLLM implements comprehensive input validation to prevent injection attacks:

### Validation Rules

1. **SQL Injection Prevention**
   ```go
   // Validates against SQL injection patterns
   sqlInjectionPatterns := []string{
       `(?i)(union\s+select)`,
       `(?i)(drop\s+table)`,
       `(?i)(delete\s+from)`,
       `(?i)(\bor\b\s+\d+\s*=\s*\d+)`,
   }
   ```

2. **XSS Prevention**
   ```go
   // HTML and script tag validation
   xssPatterns := []string{
       `<script[^>]*>.*?</script>`,
       `javascript:`,
       `on\w+\s*=`,
       `<iframe[^>]*>.*?</iframe>`,
   }
   ```

3. **Path Traversal Prevention**
   ```go
   // Path traversal validation
   pathTraversalPatterns := []string{
       `\.\.\/`,
       `\.\.\\`,
       `%2e%2e%2f`,
       `%2e%2e%5c`,
   }
   ```

### Input Sanitization

```bash
# GOLLM automatically sanitizes inputs
gollm chat "User input with <script>alert('xss')</script>"
# Script tags are sanitized before processing

# File path validation
gollm complete --file "../../../etc/passwd"
# Path traversal attempts are blocked
```

## Rate Limiting & Circuit Breakers

### Token Bucket Rate Limiting

GOLLM implements token bucket algorithm for rate limiting:

```go
// Rate limiter configuration
type RateLimiter struct {
    Capacity     int           // Bucket capacity
    RefillRate   time.Duration // Token refill rate
    TokenBucket  int           // Current tokens
    LastRefill   time.Time     // Last refill time
}

// Default configuration
rateLimiter := &RateLimiter{
    Capacity:   60,                    // 60 requests
    RefillRate: time.Minute,          // Per minute
    TokenBucket: 60,                  // Start full
}
```

#### Configuration

```yaml
# Rate limiting per provider
providers:
  openai:
    api_key: "${OPENAI_API_KEY}"
    rate_limit:
      requests_per_minute: 60
      burst_capacity: 10
      
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
    rate_limit:
      requests_per_minute: 30
      burst_capacity: 5

# Global rate limiting
rate_limiting:
  enabled: true
  global_limit: 100  # requests per minute across all providers
  per_provider_limit: true
```

### Circuit Breaker Pattern

```go
// Circuit breaker states
type CircuitState int

const (
    CircuitClosed CircuitState = iota  // Normal operation
    CircuitOpen                        // Blocking requests
    CircuitHalfOpen                    // Testing recovery
)

// Circuit breaker configuration
circuitBreaker := &CircuitBreaker{
    FailureThreshold:    5,              // Failures before opening
    RecoveryTimeout:     30 * time.Second, // Time before half-open
    SuccessThreshold:    3,              // Successes to close
}
```

#### Circuit Breaker Settings

```yaml
# Circuit breaker configuration
circuit_breaker:
  enabled: true
  failure_threshold: 5        # Failed requests before opening
  recovery_timeout: "30s"     # Time before attempting recovery
  success_threshold: 3        # Successful requests to close circuit
  timeout: "10s"             # Request timeout before failure
```

## Audit Logging

GOLLM provides comprehensive audit logging without exposing sensitive data:

### Audit Log Configuration

```yaml
# Audit logging settings
audit:
  enabled: true
  log_level: "info"
  log_file: "/var/log/gollm/audit.log"
  max_file_size: "100MB"
  max_backup_files: 10
  compress_backups: true
  
  # What to log
  log_requests: true
  log_responses: false      # Don't log response content
  log_errors: true
  log_performance: true
  
  # Security settings
  mask_credentials: true    # Always mask API keys
  mask_user_data: false    # Optional: mask user prompts
  include_stack_traces: false  # Security: don't expose internals
```

### Audit Log Format

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "info",
  "event": "api_request",
  "provider": "openai",
  "model": "gpt-3.5-turbo",
  "user": "user123",
  "session": "sess_456",
  "request_id": "req_789",
  "duration_ms": 1250,
  "tokens_used": 150,
  "success": true,
  "api_key": "sk-***...***",  // Masked
  "error": null
}
```

### Security Events

GOLLM logs security-relevant events:

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "level": "warn",
  "event": "security_validation_failed",
  "type": "input_validation",
  "rule": "xss_prevention",
  "input_hash": "sha256:abc123...",  // Hash, not actual input
  "blocked": true,
  "source_ip": "192.168.1.100"
}
```

## Environment Security

### Development Environment

```bash
# Development security setup
export GOLLM_LOG_LEVEL=debug
export GOLLM_AUDIT_ENABLED=true
export GOLLM_SECURITY_STRICT_MODE=false  # Allow localhost APIs

# Use local models for development
export GOLLM_DEFAULT_PROVIDER=ollama
export GOLLM_PROVIDERS_OLLAMA_BASE_URL=http://localhost:11434
```

### Production Environment

```bash
# Production security hardening
export GOLLM_LOG_LEVEL=warn
export GOLLM_AUDIT_ENABLED=true
export GOLLM_SECURITY_STRICT_MODE=true

# Network security
export GOLLM_TLS_MIN_VERSION=1.3
export GOLLM_CERTIFICATE_PINNING=true
export GOLLM_REQUEST_TIMEOUT=30s

# Rate limiting
export GOLLM_RATE_LIMITING_ENABLED=true
export GOLLM_RATE_LIMIT_RPM=60

# Circuit breaker
export GOLLM_CIRCUIT_BREAKER_ENABLED=true
export GOLLM_CIRCUIT_BREAKER_THRESHOLD=5
```

### Container Security

```dockerfile
# Dockerfile security best practices
FROM golang:1.21-alpine AS builder

# Create non-root user
RUN adduser -D -s /bin/sh gollm

# Build binary
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -o gollm ./cmd/gollm

# Runtime container
FROM alpine:3.18
RUN apk --no-cache add ca-certificates

# Non-root user
RUN adduser -D -s /bin/sh gollm
USER gollm

# Secure file permissions
COPY --from=builder --chown=gollm:gollm /app/gollm /usr/local/bin/gollm
RUN chmod 755 /usr/local/bin/gollm

ENTRYPOINT ["/usr/local/bin/gollm"]
```

### Kubernetes Security

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gollm
spec:
  template:
    spec:
      # Security context
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
        
      containers:
      - name: gollm
        image: gollm:latest
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
            
        # Resource limits
        resources:
          limits:
            memory: "128Mi"
            cpu: "100m"
          requests:
            memory: "64Mi"
            cpu: "50m"
            
        # Environment variables from secrets
        envFrom:
        - secretRef:
            name: gollm-secrets
            
        # Mounted volumes
        volumeMounts:
        - name: config
          mountPath: /etc/gollm
          readOnly: true
        - name: tmp
          mountPath: /tmp
          
      volumes:
      - name: config
        configMap:
          name: gollm-config
          defaultMode: 0644
      - name: tmp
        emptyDir: {}
```

## Security Testing

### Automated Security Tests

GOLLM includes comprehensive security tests:

```bash
# Run security test suite
go test -v ./internal/security -tags=security

# Specific security tests
go test -v ./internal/security -run TestCredentialHandling
go test -v ./internal/security -run TestInputValidation
go test -v ./internal/security -run TestTLSConfiguration
go test -v ./internal/security -run TestRateLimiting
go test -v ./internal/security -run TestCircuitBreaker
```

### Manual Security Validation

```bash
# Test input validation
gollm chat "'; DROP TABLE users; --"  # Should be sanitized

# Test XSS prevention  
gollm chat "<script>alert('xss')</script>"  # Should be sanitized

# Test path traversal protection
gollm complete --file "../../../../etc/passwd"  # Should be blocked

# Test rate limiting
for i in {1..100}; do gollm chat "test $i"; done  # Should be rate limited

# Test credential masking
gollm config get providers.openai.api_key  # Should show masked value
```

### Security Monitoring

```bash
# Monitor security events
tail -f /var/log/gollm/audit.log | jq 'select(.level == "warn" or .level == "error")'

# Check for failed validation attempts
grep "security_validation_failed" /var/log/gollm/audit.log

# Monitor rate limiting events
grep "rate_limit_exceeded" /var/log/gollm/audit.log

# Check circuit breaker events
grep "circuit_breaker" /var/log/gollm/audit.log
```

## Compliance

### SOC 2 Compliance

GOLLM supports SOC 2 compliance requirements:

- **Security**: Input validation, TLS encryption, access controls
- **Availability**: Circuit breakers, rate limiting, monitoring  
- **Processing Integrity**: Input validation, audit logging
- **Confidentiality**: Credential encryption, secure storage
- **Privacy**: Data minimization, audit trails

### GDPR Compliance

```yaml
# GDPR compliance settings
privacy:
  data_minimization: true        # Only collect necessary data
  encryption_at_rest: true       # Encrypt stored data
  encryption_in_transit: true    # TLS for all communications
  audit_retention: "2y"          # Audit log retention period
  user_data_retention: "30d"     # User data retention period
  
  # Right to be forgotten
  enable_data_deletion: true
  deletion_grace_period: "7d"
```

### HIPAA Compliance

```yaml
# HIPAA compliance configuration
hipaa:
  encryption_required: true
  audit_logging: true
  access_controls: true
  data_integrity: true
  
  # Technical safeguards
  tls_min_version: "1.3"
  encrypt_config: true
  secure_audit_logs: true
  
  # Administrative safeguards  
  role_based_access: true
  security_training_required: true
  incident_response_plan: true
```

## Threat Model

### Threat Analysis

| Threat | Impact | Likelihood | Mitigation |
|--------|--------|------------|------------|
| API Key Compromise | High | Medium | Secure storage, rotation, monitoring |
| Man-in-the-Middle | High | Low | TLS 1.3, certificate pinning |
| Injection Attacks | Medium | Medium | Input validation, sanitization |
| DoS Attacks | Medium | High | Rate limiting, circuit breakers |
| Credential Exposure | High | Low | Memory clearing, masked logging |
| Configuration Tampering | Medium | Low | File permissions, validation |

### Attack Vectors

1. **Network Attacks**
   - Mitigation: TLS 1.3, certificate pinning
   - Monitoring: Connection failures, certificate errors

2. **Input-Based Attacks**
   - Mitigation: Comprehensive input validation
   - Monitoring: Validation failures, suspicious patterns

3. **Credential Attacks**
   - Mitigation: Secure storage, environment variables
   - Monitoring: Authentication failures, unusual access patterns

4. **Resource Exhaustion**
   - Mitigation: Rate limiting, circuit breakers
   - Monitoring: Resource usage, request patterns

## Incident Response

### Security Incident Types

1. **API Key Compromise**
   ```bash
   # Immediate response
   gollm config set --secure providers.openai.api_key "new-secure-key"
   
   # Investigate
   grep "authentication_failed" /var/log/gollm/audit.log
   
   # Monitor
   tail -f /var/log/gollm/audit.log | grep "openai"
   ```

2. **Suspicious Activity**
   ```bash
   # Check for unusual patterns
   grep "security_validation_failed" /var/log/gollm/audit.log
   
   # Analyze source IPs
   jq '.source_ip' /var/log/gollm/audit.log | sort | uniq -c | sort -nr
   
   # Review failed requests
   jq 'select(.success == false)' /var/log/gollm/audit.log
   ```

3. **Service Disruption**
   ```bash
   # Check circuit breaker status
   gollm config get circuit_breaker.status
   
   # Review rate limiting
   grep "rate_limit_exceeded" /var/log/gollm/audit.log
   
   # Analyze performance
   jq '.duration_ms' /var/log/gollm/audit.log | awk '{sum+=$1; count++} END {print "Average:", sum/count "ms"}'
   ```

### Incident Response Playbook

1. **Detection**: Automated monitoring, manual investigation
2. **Analysis**: Log analysis, threat assessment  
3. **Containment**: Block malicious IPs, rotate credentials
4. **Eradication**: Fix vulnerabilities, update configurations
5. **Recovery**: Restore services, verify functionality
6. **Lessons Learned**: Update procedures, improve monitoring

## Best Practices

### Deployment Security

1. **Use Environment Variables for Credentials**
   ```bash
   # ✅ Good
   export OPENAI_API_KEY="sk-your-key"
   
   # ❌ Bad  
   gollm config set providers.openai.api_key "sk-your-key"  # Without --secure
   ```

2. **Enable All Security Features**
   ```yaml
   # ✅ Recommended configuration
   security:
     tls_min_version: "1.3"
     certificate_pinning: true
     input_validation: true
     
   rate_limiting:
     enabled: true
     
   circuit_breaker:
     enabled: true
     
   audit:
     enabled: true
     mask_credentials: true
   ```

3. **Regular Security Updates**
   ```bash
   # Update GOLLM regularly
   gollm version --latest
   make update
   
   # Review security configurations
   gollm config validate --security-check
   ```

### Development Security

1. **Use Secure Development Practices**
   ```bash
   # Run security tests during development
   make security-test
   
   # Validate before commits
   make lint security coverage
   
   # Use pre-commit hooks
   git config core.hooksPath .githooks
   ```

2. **Code Review Security Checklist**
   - [ ] No hardcoded credentials
   - [ ] Input validation present
   - [ ] Error handling doesn't expose sensitive data
   - [ ] TLS configuration secure
   - [ ] Rate limiting implemented
   - [ ] Audit logging included

### Operational Security

1. **Monitor Security Metrics**
   ```bash
   # Set up monitoring alerts
   # - Failed authentication attempts > 10/min
   # - Rate limiting events > 100/min
   # - Circuit breaker opens
   # - TLS handshake failures
   ```

2. **Regular Security Reviews**
   - Monthly: Review audit logs, update dependencies
   - Quarterly: Security assessment, penetration testing
   - Annually: Full security audit, compliance review

3. **Incident Response Preparation**
   - Document response procedures
   - Test incident response plans
   - Maintain emergency contact lists
   - Keep rollback procedures ready

---

## Security Resources

- **OWASP Top 10**: https://owasp.org/www-project-top-ten/
- **NIST Cybersecurity Framework**: https://www.nist.gov/cyberframework
- **CIS Controls**: https://www.cisecurity.org/controls/
- **Go Security Checklist**: https://github.com/Checkmarx/Go-SCP

For security issues or questions:
- **Security Email**: security@gollm.dev  
- **GitHub Security**: https://github.com/yourusername/gollm/security
- **Bug Bounty**: https://security.gollm.dev/bounty