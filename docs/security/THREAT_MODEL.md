# Security Dashboard - Threat Model

**Document Version:** 1.0
**Last Updated:** 2026-01-08
**Classification:** INTERNAL - SECURITY SENSITIVE

---

## 1. System Overview

The J-Expert Cyber Security Monitoring Dashboard is an **isolated security operations console** designed to provide visibility into security events, authentication failures, and privileged actions within the J-Expert recruitment platform.

### 1.1 Trust Boundaries

```
┌─────────────────────────────────────────────────────────────────────┐
│                        INTERNET (Untrusted)                         │
├─────────────────────────────────────────────────────────────────────┤
│                    CORP VPN / ALLOWLISTED IPs                       │
│                         (Trusted Network)                           │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │               SECURITY DASHBOARD (Isolated)                 │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │    │
│  │  │ IP Allowlist│→ │ TOTP MFA    │→ │ Role-Based Access   │ │    │
│  │  │  (Network)  │  │ (Identity)  │  │ OBSERVER→ANALYST→   │ │    │
│  │  │             │  │             │  │ ADMIN→DEVELOPER_ROOT│ │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘ │    │
│  └─────────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                   LOG INTEGRITY LAYER                       │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │    │
│  │  │ Hash Chain  │→ │ Merkle Root │→ │ S3 Object Lock      │ │    │
│  │  │ (Untrusted) │  │ (Untrusted) │  │ (WORM - TRUSTED)    │ │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘ │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Threat Catalog

### 2.1 Authentication Threats

| Threat ID | Threat | Mitigation | Residual Risk |
|-----------|--------|------------|---------------|
| AUTH-01 | Brute-force credential attack | Account lockout after 5 failed attempts, 15-min lockout | LOW |
| AUTH-02 | Credential stuffing | TOTP MFA required for all users | LOW |
| AUTH-03 | Session hijacking | Session bound to IP, 30-min TTL, secure cookies | LOW |
| AUTH-04 | TOTP secret compromise | Secrets stored encrypted, separate from user DB | MEDIUM |
| AUTH-05 | Replay attack on TOTP | TOTP codes valid for single 30-sec window | LOW |

### 2.2 Authorization Threats

| Threat ID | Threat | Mitigation | Residual Risk |
|-----------|--------|------------|---------------|
| AUTHZ-01 | Privilege escalation | Role fetched from DB per request, not from token | LOW |
| AUTHZ-02 | Self-approval of exports | Self-approval explicitly blocked in usecase | LOW |
| AUTHZ-03 | Unauthorized break-glass | Requires SECURITY_ADMIN base role, 50-char justification | LOW |
| AUTHZ-04 | Extended break-glass | Max 60-min hard cap, no extension, auto-revoke | LOW |

### 2.3 Log Integrity Threats

| Threat ID | Threat | Mitigation | Residual Risk |
|-----------|--------|------------|---------------|
| LOG-01 | Log tampering by admin | Hash chain makes tampering detectable | LOW |
| LOG-02 | Complete log rewrite | External S3 Object Lock anchor (WORM) | LOW |
| LOG-03 | Anchor key compromise | Object Lock GOVERNANCE mode with retention | MEDIUM |
| LOG-04 | Gap in anchoring | Daily anchor job, missing anchor = "degraded" status | LOW |

### 2.4 Network Threats

| Threat ID | Threat | Mitigation | Residual Risk |
|-----------|--------|------------|---------------|
| NET-01 | Access from untrusted network | IP allowlist blocks all non-allowlisted IPs | LOW |
| NET-02 | Allowlist bypass via proxy | Direct IP check, X-Forwarded-For validated | MEDIUM |
| NET-03 | Endpoint discovery | Non-discoverable path (noise layer), no security reliance | LOW |
| NET-04 | DoS on security console | Rate limiting, separate from main app | LOW |

### 2.5 Insider Threats

| Threat ID | Threat | Mitigation | Residual Risk |
|-----------|--------|------------|---------------|
| INSIDER-01 | Malicious SECURITY_ADMIN | All actions logged, export requires justification | MEDIUM |
| INSIDER-02 | Break-glass abuse | Time-limited, auto-revoke, CRITICAL severity logging | LOW |
| INSIDER-03 | Log deletion by DBA | External anchor in S3 Object Lock | LOW |

---

## 3. Attack Scenarios

### Scenario 1: Compromised SOC Analyst Credential

**Attack Path:**
1. Attacker obtains SOC analyst username/password
2. Attacker attempts login from external network

**Mitigation Chain:**
1. ❌ IP allowlist blocks (not on VPN)
2. ❌ Even if on VPN, TOTP required
3. ❌ All login attempts logged

**Verdict:** Attack blocked at network layer

### Scenario 2: Insider Attempts Log Cover-up

**Attack Path:**
1. SECURITY_ADMIN performs unauthorized action
2. SECURITY_ADMIN attempts to delete evidence

**Mitigation Chain:**
1. ✓ Action logged immediately with hash chain
2. ✓ Daily anchor already in S3 Object Lock
3. ✓ Deletion detected via chain break verification
4. ✓ Chain break event = CRITICAL severity

**Verdict:** Tampering detected, external evidence preserved

### Scenario 3: Break-Glass Abuse

**Attack Path:**
1. SECURITY_ADMIN activates break-glass for personal investigation
2. Uses elevated privileges inappropriately

**Mitigation Chain:**
1. ✓ Must provide 50+ char justification
2. ✓ Activation logged as CRITICAL severity
3. ✓ Auto-expires in max 60 minutes
4. ✓ All actions during elevated session logged
5. ✓ Post-incident review of break-glass sessions

**Verdict:** Abuse visible in audit trail

---

## 4. Security Controls Matrix

| Control | Implementation | Verification |
|---------|---------------|--------------|
| IP Allowlist | `SecurityIPAllowlistMiddleware` | Blocked requests logged |
| TOTP MFA | `pquerna/otp`, enforced for all users | Login flow requires TOTP |
| Session Binding | IP+UserAgent, 30-min TTL | Mismatch = session invalid |
| Role-Based Access | Per-endpoint middleware | Role hierarchy enforced |
| Break-Glass Limit | 60-min CHECK constraint in DB | Extension not possible |
| Hash Chain | SHA-256, previous_hash column | Verification API endpoint |
| External Anchor | S3 Object Lock GOVERNANCE mode | WORM, 1-year retention |
| Audit Logging | All actions via SecurityLogger | Persisted to security_events |

---

## 5. Residual Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| TOTP secret database breach | LOW | HIGH | Encrypt secrets at rest |
| S3 credentials compromise | LOW | HIGH | Rotate keys, minimal permissions |
| X-Forwarded-For spoofing | MEDIUM | MEDIUM | Validate reverse proxy config |
| Single point of failure (S3) | LOW | MEDIUM | Consider multi-anchor strategy |

---

## 6. Recommendations

1. **Implement TOTP secret encryption** at rest using a separate key
2. **Configure S3 bucket policy** to restrict access to anchor-writer role only
3. **Set up alerting** for CRITICAL severity events
4. **Establish break-glass review process** for post-incident analysis
5. **Rotate security user credentials** every 90 days
