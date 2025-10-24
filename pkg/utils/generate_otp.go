package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateSecureOTP generates a secure random 6-digit OTP (100000–999999)
func GenerateSecureOTP() (string, error) {
	min := int64(100000)
	max := int64(999999)

	nBig, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return "", err
	}

	otp := nBig.Int64() + min
	return fmt.Sprintf("%d", otp), nil
}
