package node

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

func GenerateAPIKey(nodeID string) (string, error) {
	secret := make([]byte, 24)

	if _, err := rand.Read(secret); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"%s.%s",
		nodeID,
		base64.RawURLEncoding.EncodeToString(secret),
	), nil
}

func GenerateNodeID() string {
	return uuid.New().String()
}
