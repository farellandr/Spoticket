package helpers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

func deriveKey(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:]
}

var secretKey = deriveKey(os.Getenv("JWT_SECRET"))

func EncryptExternalID(ticketID uuid.UUID, couponID *uuid.UUID) string {
	plaintext := ticketID.String()
	if couponID != nil {
		plaintext = fmt.Sprintf("%s|%s", ticketID.String(), couponID.String())
	}

	block, _ := aes.NewCipher(secretKey)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.URLEncoding.EncodeToString(ciphertext)
}

func DecryptExternalID(encrypted string) (ticketID uuid.UUID, couponID *uuid.UUID, err error) {
	data, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return uuid.Nil, nil, err
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return uuid.Nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return uuid.Nil, nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return uuid.Nil, nil, fmt.Errorf("invalid cipher text")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return uuid.Nil, nil, err
	}

	parts := strings.Split(string(plaintext), "|")
	if len(parts) == 1 {
		ticketID, err := uuid.Parse(parts[0])
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("invalid ticket ID format")
		}
		return ticketID, nil, nil
	}

	if len(parts) == 2 {
		ticketID, err := uuid.Parse(parts[0])
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("invalid ticket ID format")
		}

		parsedCouponID, err := uuid.Parse(parts[1])
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("invalid coupon ID format")
		}
		return ticketID, &parsedCouponID, nil
	}

	return uuid.Nil, nil, fmt.Errorf("invalid plaintext format")
}

func ExtractTicketID(externalID string) (uuid.UUID, *uuid.UUID, error) {
	parts := strings.Split(externalID, "-")
	if len(parts) < 3 {
		return uuid.Nil, nil, fmt.Errorf("invalid external ID format")
	}

	encryptedPart := strings.Join(parts[2:], "-")

	return DecryptExternalID(encryptedPart)
}
