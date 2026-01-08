# Security Dashboard - Log Review Scenarios

**Document Version:** 1.0
**Last Updated:** 2026-01-08
**Purpose:** SOC Operator Training & Incident Response Reference

---

## Scenario 1: Brute-Force Attack Detection

### Indicators
- Multiple `login_failed` events from same IP within short timeframe
- `login_blocked` event following failed attempts
- Severity escalation: WARN → HIGH

### Investigation Steps
1. Filter events by IP address in `/events` page
2. Check heatmap for attack pattern/timing
3. Review user agent strings for automation indicators
4. Cross-reference with `rate_limit_triggered` events

### Dashboard Query
```
Filters:
- Severity: WARN, HIGH
- Event Types: login_failed, login_blocked, rate_limit_triggered
- Time Range: Last 24 hours
```

### Response Actions
- Verify block is active (auto-15min)
- Consider permanent IP blocklist if automated attack
- Notify affected user if legitimate account targeted

---

## Scenario 2: Credential Stuffing Investigation

### Indicators
- `login_failed` events across MULTIPLE user accounts
- Same source IP or small IP range
- User agents consistent with attack tools

### Investigation Steps
1. Open heatmap view, look for concentrated failure periods
2. Group events by IP in events view
3. Check if failures target valid vs invalid usernames
4. Review time distribution (automated = consistent intervals)

### Dashboard Query
```
Filters:
- Event Type: login_failed
- Group By: IP Address
- Sort: Event Count DESC
```

### Response Actions
- Block offending IP ranges at firewall level
- Enable enhanced monitoring for targeted accounts
- Consider temporary account lockdown for highly-targeted users

---

## Scenario 3: Privilege Escalation Attempt

### Indicators
- `unauthorized_access` events for admin endpoints
- `role_modified` events not initiated by SECURITY_ADMIN
- Anomalous access patterns from authenticated users

### Investigation Steps
1. Filter by `unauthorized_access` event type
2. Identify the user attempting elevation
3. Review user's recent activity timeline
4. Check if account may be compromised

### Dashboard Query
```
Filters:
- Event Types: unauthorized_access, role_modified
- Severity: HIGH, CRITICAL
```

### Response Actions
- Disable user account immediately
- Review all actions taken by user in last 24 hours
- Initiate incident response if compromise confirmed
- Document for post-incident review

---

## Scenario 4: Break-Glass Session Review

### Indicators
- `breakglass_activated` events (CRITICAL severity)
- Elevated privilege actions during session window
- `breakglass_expired` or `breakglass_revoked` events

### Investigation Steps
1. Identify the break-glass session from timeline view
2. Review the justification provided
3. List ALL actions taken during the session window
4. Verify actions were consistent with stated justification

### Dashboard Query
```
Filters:
- Event Types: breakglass_activated, breakglass_expired, breakglass_revoked
- Time Range: Filter to session window
```

### Post-Incident Questions
- Was the justification appropriate for actions taken?
- Were any sensitive data accessed or exported?
- Should the operator's access be modified?

---

## Scenario 5: Log Integrity Alert

### Indicators
- `hash_chain_break` event (CRITICAL severity)
- Integrity status showing "compromised"
- Missing or mismatched anchors

### Investigation Steps
1. Check integrity status in dashboard header
2. Run integrity verification for affected date range
3. Identify first event with chain break
4. Compare against S3 anchored hashes

### Dashboard Query
```
Filters:
- Event Type: hash_chain_break
- Severity: CRITICAL
```

### Response Actions
- **CRITICAL**: This indicates potential tampering
- Preserve S3 anchor data as evidence
- Initiate forensic investigation
- Consider involving external security team
- Document chain of custody for all evidence

---

## Scenario 6: Data Export Audit

### Indicators
- `data_export` request events
- `data_export_approved` or `data_export_rejected` events
- Download activity for approved exports

### Investigation Steps
1. Review all pending export requests
2. Verify justifications are appropriate
3. Check download counts on approved exports
4. Ensure approver ≠ requester

### Dashboard Query
```
Filters:
- Event Types: data_export, data_export_approved, data_export_rejected
- Group By: Requester
```

### Compliance Questions
- Was export justified by business need?
- Was data minimization applied (filtered scope)?
- Is download count reasonable?

---

## Scenario 7: Suspicious Input Detection

### Indicators
- `suspicious_input` events
- `csrf_violation` events
- `validation_failed` with injection patterns

### Investigation Steps
1. Filter events by `suspicious_input`
2. Review the input details in event payload
3. Check if same source IP is involved
4. Correlate with any successful requests

### Dashboard Query
```
Filters:
- Event Types: suspicious_input, csrf_violation
- Severity: HIGH
```

### Response Actions
- Block source IP if attack confirmed
- Review application logs for exploitation success
- Update WAF rules if applicable
- Document attack vectors for future prevention

---

## Quick Reference: Severity Escalation

| Severity | Action Required | Response Time |
|----------|-----------------|---------------|
| INFO | Monitor only | N/A |
| MEDIUM | Review in daily triage | 24 hours |
| WARN | Investigate same day | 8 hours |
| HIGH | Investigate immediately | 1 hour |
| CRITICAL | Incident response activation | 15 minutes |

---

## Contact Escalation

- **L1 SOC Analyst**: Initial triage and investigation
- **L2 Security Engineer**: Incident confirmation and response
- **Security Manager**: Breach notification decisions
- **Legal/Compliance**: Regulatory notification requirements
