-- Thêm cột google_id để hỗ trợ đăng nhập Google OAuth
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS google_id VARCHAR(255) UNIQUE;
-- Cho phép password_hash NULL (tài khoản Google không cần password)
ALTER TABLE accounts ALTER COLUMN password_hash DROP NOT NULL;
