package helpers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"time"

	"github.com/google/uuid"
)

func NewDokuHeaderGenerator(clientID, secretKey, requestPath string) *DokuHeaderGenerator {
	return &DokuHeaderGenerator{
		ClientID:    clientID,
		SecretKey:   secretKey,
		RequestID:   uuid.New().String(),
		RequestPath: requestPath,
	}
}

type DokuHeaderGenerator struct {
	ClientID    string
	SecretKey   string
	RequestID   string
	RequestPath string
}

func (d *DokuHeaderGenerator) GenerateDigest(jsonBody string) string {
	hash := sha256.Sum256([]byte(jsonBody))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (d *DokuHeaderGenerator) GenerateSignature(digest string) string {
	requestTimestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	componentSignature := "Client-Id:" + d.ClientID + "\n" +
		"Request-Id:" + d.RequestID + "\n" +
		"Request-Timestamp:" + requestTimestamp + "\n" +
		"Request-Target:" + d.RequestPath + "\n" +
		"Digest:" + digest

	mac := hmac.New(sha256.New, []byte(d.SecretKey))
	mac.Write([]byte(componentSignature))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return "HMACSHA256=" + signature
}

func (d *DokuHeaderGenerator) GetHeaders(jsonBody string) map[string]string {
	digest := d.GenerateDigest(jsonBody)
	signature := d.GenerateSignature(digest)
	requestTimestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	return map[string]string{
		"Client-Id":         d.ClientID,
		"Request-Id":        d.RequestID,
		"Request-Timestamp": requestTimestamp,
		"Signature":         signature,
		"Content-Type":      "application/json",
		"Digest":            digest,
	}
}
