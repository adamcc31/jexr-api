# Panduan Deployment Security Dashboard

**Dokumen:** Checklist Deployment Step-by-Step
**Tanggal:** 8 Januari 2026
**Status:** Siap untuk produksi

---

## 📋 Checklist Deployment

### Tahap 1: Persiapan Database

- [ ] **1.1 Jalankan migrasi database**
  ```bash
  cd jexpert-backend
  migrate -path ./migrations -database "$DATABASE_URL" up
  ```
  > Pastikan migrasi 000020_security_dashboard berhasil

- [ ] **1.2 Verifikasi tabel terbuat**
  ```sql
  SELECT table_name FROM information_schema.tables 
  WHERE table_name LIKE 'security_%' OR table_name LIKE '%_glass%' OR table_name LIKE 'allowed_%';
  ```
  > Harus ada 6 tabel: `security_users`, `security_sessions`, `break_glass_sessions`, `allowed_ip_ranges`, `hash_anchors`, `export_requests`

---

### Tahap 2: Konfigurasi Jaringan

- [ ] **2.1 Tentukan IP allowlist untuk SOC team**
  
  Edit file `scripts/seed_security_dashboard.sql`:
  ```sql
  INSERT INTO allowed_ip_ranges (cidr, description, is_active) VALUES
      ('YOUR_VPN_CIDR', 'VPN SOC Team', true),
      ('YOUR_OFFICE_IP/32', 'IP Kantor SOC', true);
  ```

- [ ] **2.2 HAPUS localhost dari allowlist (produksi)**
  ```sql
  -- Jangan sertakan baris ini di produksi:
  -- ('127.0.0.1/32', 'Localhost', true)
  ```

---

### Tahap 3: Buat User Security Awal

- [ ] **3.1 Generate password hash baru**
  ```bash
  # Linux/Mac:
  htpasswd -nbBC 10 "" "PasswordKuatAnda123!" | tr -d ':\n' | sed 's/$2y/$2a/'
  
  # Atau gunakan online bcrypt generator dengan cost 10+
  ```

- [ ] **3.2 Update password hash di seed script**
  
  Edit `scripts/seed_security_dashboard.sql`:
  ```sql
  INSERT INTO security_users (..., password_hash, ...) VALUES (
      ...,
      '$2a$10$HASH_BARU_ANDA_DISINI',
      ...
  );
  ```

- [ ] **3.3 Jalankan seed script**
  ```bash
  psql $DATABASE_URL -f scripts/seed_security_dashboard.sql
  ```

- [ ] **3.4 Verifikasi user terbuat**
  ```sql
  SELECT username, role, totp_enabled FROM security_users;
  ```

---

### Tahap 4: Konfigurasi Wasabi Object Lock

- [ ] **4.1 Login ke Wasabi Console**
  - Buka [console.wasabisys.com](https://console.wasabisys.com)
  - Login dengan akun Wasabi Anda

- [ ] **4.2 Buat bucket dengan Object Lock**
  1. Klik **Create Bucket**
  2. Nama bucket: `jexpert-security-anchors`
  3. Region: `ap-southeast-1` (Singapore) atau sesuai lokasi
  4. ✅ **Centang: Enable Object Lock** ⚠️ WAJIB!
  5. Klik **Create Bucket**

  > ⚠️ **PENTING:** Object Lock HARUS diaktifkan saat buat bucket. Tidak bisa diaktifkan setelahnya!

- [ ] **4.3 Buat Access Key baru**
  1. Klik **Access Keys** di sidebar
  2. Klik **Create New Access Key**
  3. Simpan Access Key ID dan Secret Access Key

- [ ] **4.4 (Opsional) Konfigurasi via CLI**
  ```bash
  # Set Wasabi profile
  aws configure --profile wasabi
  # Access Key ID: [your-key]
  # Secret Access Key: [your-secret]
  # Region: ap-southeast-1

  # Set retention policy
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

### Tahap 5: Set Environment Variables

- [ ] **5.1 Tambahkan ke .env produksi**
  ```env
  # Wasabi Configuration
  S3_PROVIDER=wasabi
  S3_ACCESS_KEY_ID=your-wasabi-access-key
  S3_SECRET_ACCESS_KEY=your-wasabi-secret-key
  S3_REGION=ap-southeast-1
  SECURITY_ANCHOR_BUCKET=jexpert-security-anchors
  
  # Security Dashboard path
  SECURITY_DASHBOARD_PATH=sec-console-rahasia
  
  # Session config
  SECURITY_SESSION_TTL_MINUTES=30
  SECURITY_MAX_LOGIN_ATTEMPTS=5
  ```

- [ ] **5.2 Restart aplikasi backend**
  ```bash
  # Railway/Docker:
  railway up
  
  # Atau manual:
  go build -o api ./cmd/api && ./api
  ```

---

### Tahap 6: Setup TOTP untuk Semua User

TOTP setup dilakukan via API, bukan frontend karena user belum bisa login tanpa TOTP.

- [ ] **6.1 Panggil endpoint setup-totp untuk generate secret**
  ```bash
  # Ganti dengan IP yang ada di allowlist
  curl -X POST https://api.domain.com/v1/{SECURITY_DASHBOARD_PATH}/auth/setup-totp \
    -H "Content-Type: application/json" \
    -d '{"username": "secadmin", "password": "PasswordAnda123!"}' 
  ```
  
  Response akan berisi:
  ```json
  {
    "data": {
      "secret": "JBSWY3DPEHPK3PXP",
      "qrCodeUrl": "otpauth://totp/J-Expert%20Security:secadmin?secret=JBSWY3DPEHPK3PXP&issuer=J-Expert%20Security",
      "setupGuide": "Scan QR code dengan Google Authenticator..."
    }
  }
  ```

- [ ] **6.2 Tambahkan ke Authenticator App**
  
  **Opsi A: QR Code**
  - Buka URL di browser: `https://api.qrserver.com/v1/create-qr-code/?data={qrCodeUrl}`
  - Scan dengan Google Authenticator, Authy, atau 1Password
  
  **Opsi B: Manual**
  - Di authenticator app, pilih "Enter manually"
  - Account: `secadmin`  
  - Key: `JBSWY3DPEHPK3PXP` (dari response secret)

- [ ] **6.3 Konfirmasi TOTP dengan kode dari app**
  ```bash
  curl -X POST https://api.domain.com/v1/{SECURITY_DASHBOARD_PATH}/auth/confirm-totp \
    -H "Content-Type: application/json" \
    -d '{"username": "secadmin", "password": "PasswordAnda123!", "totpCode": "123456"}'
  ```
  
  Response sukses:
  ```json
  {"message": "TOTP berhasil diaktifkan! Silakan login kembali."}
  ```

- [ ] **6.4 Verifikasi TOTP aktif di database**
  ```sql
  SELECT username, totp_enabled FROM security_users WHERE totp_enabled = true;
  ```

- [ ] **6.5 Ulangi langkah 6.1-6.3 untuk user lain** (secanalyst, secobserver)

---

### Tahap 7: Testing & Verifikasi

- [ ] **7.1 Test akses dari IP non-allowlist**
  > Harus mendapat response 403 Forbidden

- [ ] **7.2 Test login dengan password salah (5x)**
  > User harus ter-lock selama 15 menit

- [ ] **7.3 Test login berhasil dengan TOTP**
  > Harus redirect ke dashboard

- [ ] **7.4 Test role-based access**
  - Observer: hanya bisa lihat, tidak bisa export
  - Analyst: bisa request export
  - Admin: bisa approve export, aktifkan break-glass

- [ ] **7.5 Verifikasi integrity status**
  > Dashboard harus menunjukkan status "INTACT" atau "DEGRADED"

---

### Tahap 8: Setup Cron Job (Opsional)

- [ ] **8.1 Buat daily anchor job**
  ```bash
  # Crontab: setiap hari jam 00:05 UTC
  5 0 * * * curl -X POST https://api.domain.com/v1/{PATH}/internal/anchor-daily
  ```

- [ ] **8.2 Buat log cleanup job**
  ```sql
  -- Di PostgreSQL dengan pg_cron:
  SELECT cron.schedule('cleanup-old-logs', '0 3 * * *', 
      $$SELECT cleanup_old_security_events(90, 1000, 100)$$);
  ```

---

## ⚠️ Catatan Penting

1. **JANGAN** deploy tanpa mengubah password default
2. **JANGAN** sertakan localhost di allowlist produksi
3. **WAJIB** enable TOTP sebelum akses ke data sensitif
4. **BACKUP** database sebelum menjalankan migrasi
5. **DOKUMENTASIKAN** semua IP yang di-allowlist

---

## 📞 Kontak Darurat

Jika terjadi masalah saat deployment:
- Rollback migrasi: `migrate -path ./migrations -database "$DATABASE_URL" down 1`
- Restore dari backup yang dibuat sebelumnya
