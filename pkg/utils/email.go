package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/gomail.v2"
)

func SendEmail(to, subject, body string, attachments ...string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASS")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	smtpPort, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid SMTP_PORT: %v", err)
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", from)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body)

	for _, filePath := range attachments {
		if _, err := os.Stat(filePath); err != nil {
			Logger.Warnf("Attachment not found, skipping: %s", filePath)
			continue
		}
		msg.Attach(filePath, gomail.Rename(filepath.Base(filePath)))
	}

	d := gomail.NewDialer(host, smtpPort, from, password)
	if err := d.DialAndSend(msg); err != nil {
		Logger.Errorf("failed to send email to %s", to)
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
