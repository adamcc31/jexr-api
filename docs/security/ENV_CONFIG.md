# Security Dashboard Environment Configuration

**Mendukung:** AWS S3 dan **Wasabi** (S3-compatible)

---

## S3-Compatible Storage (Wasabi)

```env
# ============================================
# WASABI CONFIGURATION (Recommended)
# ============================================

# Set provider ke wasabi
S3_PROVIDER=wasabi

# Wasabi credentials (dari Wasabi Console > Access Keys)
S3_ACCESS_KEY_ID=your-wasabi-access-key
S3_SECRET_ACCESS_KEY=your-wasabi-secret-key

# Region Wasabi (lihat daftar di bawah)
S3_REGION=ap-southeast-1

# Bucket name (harus sudah dibuat dengan Object Lock)
SECURITY_ANCHOR_BUCKET=jexpert-security-anchors

# Optional: Custom endpoint (default otomatis berdasarkan region)
# WASABI_ENDPOINT=s3.ap-southeast-1.wasabisys.com
```

### Wasabi Region Endpoints

| Region | Endpoint |
|--------|----------|
| `us-east-1` | `s3.us-east-1.wasabisys.com` |
| `us-east-2` | `s3.us-east-2.wasabisys.com` |
| `us-west-1` | `s3.us-west-1.wasabisys.com` |
| `eu-central-1` | `s3.eu-central-1.wasabisys.com` |
| `eu-west-1` | `s3.eu-west-1.wasabisys.com` |
| `eu-west-2` | `s3.eu-west-2.wasabisys.com` |
| `ap-northeast-1` | `s3.ap-northeast-1.wasabisys.com` (Tokyo) |
| `ap-northeast-2` | `s3.ap-northeast-2.wasabisys.com` (Osaka) |
| `ap-southeast-1` | `s3.ap-southeast-1.wasabisys.com` (Singapore) |
| `ap-southeast-2` | `s3.ap-southeast-2.wasabisys.com` (Sydney) |

---

## Membuat Bucket Wasabi dengan Object Lock

### Via Wasabi Console

1. Login ke [Wasabi Console](https://console.wasabisys.com)
2. Klik **Create Bucket**
3. Nama bucket: `jexpert-security-anchors`
4. Region: `ap-southeast-1` (atau sesuai lokasi)
5. **PENTING:** Centang ✅ **Enable Object Lock**
6. Klik **Create Bucket**

### Via AWS CLI (dengan Wasabi endpoint)

```bash
# Set Wasabi profile
aws configure --profile wasabi
# Access Key ID: [your-wasabi-key]
# Secret Access Key: [your-wasabi-secret]
# Region: ap-southeast-1
# Output: json

# Create bucket dengan Object Lock
aws s3api create-bucket \
    --bucket jexpert-security-anchors \
    --region ap-southeast-1 \
    --object-lock-enabled-for-bucket \
    --endpoint-url https://s3.ap-southeast-1.wasabisys.com \
    --profile wasabi

# Set default retention (GOVERNANCE mode, 1 tahun)
aws s3api put-object-lock-configuration \
    --bucket jexpert-security-anchors \
    --object-lock-configuration '{
        "ObjectLockEnabled": "Enabled",
        "Rule": {
            "DefaultRetention": {
                "Mode": "GOVERNANCE",
                "Years": 1
            }
        }
    }' \
    --endpoint-url https://s3.ap-southeast-1.wasabisys.com \
    --profile wasabi
```

---

## Security Dashboard Configuration

```env
# ============================================
# SECURITY DASHBOARD PATH
# ============================================

# Path dashboard (noise layer, bukan security control)
SECURITY_DASHBOARD_PATH=sec-console-internal

# ============================================
# SESSION CONFIGURATION
# ============================================

SECURITY_SESSION_TTL_MINUTES=30
SECURITY_MAX_LOGIN_ATTEMPTS=5
SECURITY_LOCKOUT_DURATION_MINUTES=15

# ============================================
# BREAK-GLASS CONFIGURATION
# ============================================

SECURITY_BREAKGLASS_MAX_MINUTES=60
SECURITY_BREAKGLASS_MIN_JUSTIFICATION=50
```

---

## Perbedaan Wasabi vs AWS S3

| Fitur | AWS S3 | Wasabi |
|-------|--------|--------|
| Object Lock | ✅ Supported | ✅ Supported |
| GOVERNANCE Mode | ✅ | ✅ |
| COMPLIANCE Mode | ✅ | ✅ |
| Egress Fees | Berbayar | **Gratis** |
| Request Pricing | Per-request | Termasuk |
| Path-Style URLs | Optional | **Required** |

> **Catatan:** Wasabi mengharuskan path-style URLs, sudah dikonfigurasi di `s3_client.go`

---

## Troubleshooting Wasabi

### Error: "The bucket does not allow object locks"
- Bucket harus dibuat dengan Object Lock **enabled dari awal**
- Object Lock tidak bisa diaktifkan setelah bucket dibuat
- Solusi: Buat bucket baru dengan Object Lock enabled

### Error: "Access Denied"
- Periksa Access Key dan Secret Key
- Pastikan bucket policy mengizinkan akses dari key tersebut
- Periksa region endpoint sesuai dengan lokasi bucket

### Error: "SignatureDoesNotMatch"
- Pastikan tidak ada spasi di credentials
- Periksa format endpoint (gunakan HTTPS)
- Pastikan region sesuai

---

## Production Checklist

- [ ] Buat Wasabi Access Key dengan permission minimal
- [ ] Buat bucket dengan Object Lock enabled
- [ ] Test koneksi dengan `TestS3Connection()`
- [ ] Set environment variables di server
- [ ] Verifikasi hash anchor tersimpan setelah test

