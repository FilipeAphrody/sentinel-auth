package security

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/url"

	"github.com/pquerna/otp/totp"
)

// GenerateMFASecret generates a random Base32 string (compatible with TOTP secrets).
func GenerateMFASecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	// Google Authenticator requires Base32, not Base64
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GetMFAQRCodeURI returns the URI for QR code generation (compatible with Google Authenticator).
func GetMFAQRCodeURI(email, secret string) string {
	issuer := "SentinelAuth"
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		url.PathEscape(issuer), url.PathEscape(email), secret, url.QueryEscape(issuer))
}

// VerifyMFACode checks if the provided 6-digit code is valid for the given secret.
func VerifyMFACode(code, secret string) bool {
	return totp.Validate(code, secret)
}