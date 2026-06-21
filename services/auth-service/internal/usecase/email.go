package usecase

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strconv"
	"strings"
)

// EmailService gửi email qua SMTP (Gmail hoặc bất kỳ provider nào)
type EmailService struct {
	host        string
	port        int
	username    string
	password    string
	from        string
	frontendURL string
}

func NewEmailService() *EmailService {
	port, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	if port == 0 {
		port = 587
	}
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4200"
	}
	return &EmailService{
		host:        os.Getenv("SMTP_HOST"),
		port:        port,
		username:    os.Getenv("SMTP_USER"),
		password:    os.Getenv("SMTP_PASSWORD"),
		from:        os.Getenv("SMTP_FROM"),
		frontendURL: frontendURL,
	}
}

// IsConfigured kiểm tra xem SMTP đã được cấu hình chưa
func (e *EmailService) IsConfigured() bool {
	return e.host != "" && e.username != "" && e.password != ""
}

// SendVerificationEmail gửi email xác thực tài khoản
func (e *EmailService) SendVerificationEmail(toEmail, username, verifyURL string) error {
	if !e.IsConfigured() {
		// Dev mode: chỉ log ra console
		fmt.Printf("[EMAIL-DEV] Gửi xác thực tới %s: %s\n", toEmail, verifyURL)
		return nil
	}

	subject := "WorkTogether – Xác thực địa chỉ email của bạn"
	body := buildVerificationEmailHTML(username, verifyURL)
	return e.sendHTML(toEmail, subject, body)
}

// SendWelcomeEmail gửi email chào mừng sau khi xác thực thành công
func (e *EmailService) SendWelcomeEmail(toEmail, username string) error {
	if !e.IsConfigured() {
		fmt.Printf("[EMAIL-DEV] Chào mừng %s (%s)\n", username, toEmail)
		return nil
	}

	subject := "Chào mừng đến với WorkTogether! 🎵"
	body := buildWelcomeEmailHTML(username, e.frontendURL)
	return e.sendHTML(toEmail, subject, body)
}

func (e *EmailService) SendPasswordResetEmail(toEmail, username, resetURL string) error {
	if !e.IsConfigured() {
		fmt.Printf("[EMAIL-DEV] Gửi link reset mật khẩu tới %s: %s\n", toEmail, resetURL)
		return nil
	}
	subject := "WorkTogether – Khôi phục mật khẩu của bạn"
	body := fmt.Sprintf(`<h2>Xin chào, %s!</h2><p>Nhấp vào liên kết sau để đặt lại mật khẩu: <a href="%s">%s</a></p>`, username, resetURL, resetURL)
	return e.sendHTML(toEmail, subject, body)
}


func (e *EmailService) sendHTML(to, subject, htmlBody string) error {
	fromHeader := fmt.Sprintf("WorkTogether <%s>", e.from)
	headers := strings.Join([]string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
	}, "\r\n")
	message := headers + "\r\n\r\n" + htmlBody

	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)

	// STARTTLS (port 587)
	if e.port == 587 {
		conn, err := smtp.Dial(addr)
		if err != nil {
			return fmt.Errorf("smtp dial: %w", err)
		}
		defer conn.Close()

		if err := conn.StartTLS(&tls.Config{ServerName: e.host}); err != nil {
			return fmt.Errorf("starttls: %w", err)
		}
		if err := conn.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
		if err := conn.Mail(e.from); err != nil {
			return fmt.Errorf("smtp mail: %w", err)
		}
		if err := conn.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
		w, err := conn.Data()
		if err != nil {
			return fmt.Errorf("smtp data: %w", err)
		}
		_, err = fmt.Fprint(w, message)
		if err != nil {
			return err
		}
		return w.Close()
	}

	// SSL (port 465)
	tlsConn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: e.host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer tlsConn.Close()

	client, err := smtp.NewClient(tlsConn, e.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(e.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, message)
	if err != nil {
		return err
	}
	return w.Close()
}

func buildVerificationEmailHTML(username, verifyURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#0d0d0d;font-family:'Segoe UI',sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0">
    <tr><td align="center" style="padding:40px 20px;">
      <table width="560" style="background:#141414;border-radius:12px;border:1px solid #222;overflow:hidden;">
        <!-- Header -->
        <tr><td style="background:linear-gradient(135deg,#7c3aed,#4f46e5);padding:32px;text-align:center;">
          <span style="font-size:24px;font-weight:800;color:#fff;letter-spacing:-0.5px;">WorkTogether</span>
          <p style="color:rgba(255,255,255,0.8);margin:8px 0 0;font-size:14px;">Nền tảng âm nhạc & làm việc nhóm</p>
        </td></tr>
        <!-- Body -->
        <tr><td style="padding:40px 32px;">
          <h2 style="color:#fff;margin:0 0 12px;font-size:22px;">Xin chào, %s! 👋</h2>
          <p style="color:#9ca3af;line-height:1.6;margin:0 0 28px;">Cảm ơn bạn đã đăng ký WorkTogether. Nhấn nút bên dưới để xác thực địa chỉ email và bắt đầu trải nghiệm.</p>
          <div style="text-align:center;margin:32px 0;">
            <a href="%s" style="display:inline-block;background:linear-gradient(135deg,#7c3aed,#4f46e5);color:#fff;text-decoration:none;padding:14px 36px;border-radius:8px;font-weight:600;font-size:16px;letter-spacing:0.3px;">
              ✅ Xác thực Email
            </a>
          </div>
          <p style="color:#6b7280;font-size:13px;line-height:1.5;">Liên kết hết hạn sau <strong style="color:#9ca3af;">24 giờ</strong>. Nếu bạn không đăng ký tài khoản này, hãy bỏ qua email này.</p>
        </td></tr>
        <!-- Footer -->
        <tr><td style="background:#0d0d0d;padding:20px 32px;text-align:center;border-top:1px solid #222;">
          <p style="color:#4b5563;font-size:12px;margin:0;">© 2026 WorkTogether • Thiết kế tối giản, tối ưu năng suất</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, username, verifyURL)
}

func buildWelcomeEmailHTML(username, frontendURL string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="margin:0;padding:0;background:#0d0d0d;font-family:'Segoe UI',sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0">
    <tr><td align="center" style="padding:40px 20px;">
      <table width="560" style="background:#141414;border-radius:12px;border:1px solid #222;overflow:hidden;">
        <tr><td style="background:linear-gradient(135deg,#7c3aed,#4f46e5);padding:32px;text-align:center;">
          <span style="font-size:24px;font-weight:800;color:#fff;">WorkTogether</span>
        </td></tr>
        <tr><td style="padding:40px 32px;text-align:center;">
          <div style="font-size:48px;margin-bottom:16px;">🎵</div>
          <h2 style="color:#fff;margin:0 0 12px;">Chào mừng, %s!</h2>
          <p style="color:#9ca3af;line-height:1.6;">Tài khoản của bạn đã được xác thực. Bạn có thể đăng nhập và bắt đầu ngay bây giờ.</p>
          <a href="%s/auth" style="display:inline-block;margin-top:24px;background:linear-gradient(135deg,#7c3aed,#4f46e5);color:#fff;text-decoration:none;padding:14px 36px;border-radius:8px;font-weight:600;">
            Vào WorkTogether →
          </a>
        </td></tr>
        <tr><td style="background:#0d0d0d;padding:20px 32px;text-align:center;border-top:1px solid #222;">
          <p style="color:#4b5563;font-size:12px;margin:0;">© 2026 WorkTogether</p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, username, frontendURL)
}
