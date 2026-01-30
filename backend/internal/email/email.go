package email

import (
	"fmt"
	"net/smtp"
	"rpms-backend/internal/config"
)

type EmailSender struct {
	config *config.Config
}

func NewEmailSender(cfg *config.Config) *EmailSender {
	return &EmailSender{config: cfg}
}

func (s *EmailSender) SendVerificationEmail(toEmail, code string) error {
	// If SMTP credentials are not set, fallback to logging (or return error)
	if s.config.SMTP.Email == "" || s.config.SMTP.Password == "" {
		fmt.Printf("SMTP credentials not set. Mocking email to %s with code %s\n", toEmail, code)
		return nil
	}

	from := s.config.SMTP.Email
	password := s.config.SMTP.Password
	host := s.config.SMTP.Host
	port := s.config.SMTP.Port
	address := host + ":" + port

	subject := "Subject: Verify your RPMS Account\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
		<html>
			<body>
				<h2>Welcome to RPMS!</h2>
				<p>Please use the following code to verify your email address:</p>
				<h1>%s</h1>
				<p>If you did not request this, please ignore this email.</p>
			</body>
		</html>
	`, code)

	message := []byte(subject + mime + body)

	auth := smtp.PlainAuth("", from, password, host)

	err := smtp.SendMail(address, auth, from, []string{toEmail}, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
