package node

import (
	"context"
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	DB *sql.DB
}

func (r *Repository) CreateNode(
	ctx context.Context,
	nodeID string,
	apiKey string,
) error {
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(apiKey),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return err
	}

	_, err = r.DB.ExecContext(
		ctx,
		`INSERT INTO nodes (node_id, api_key_hash)
		 VALUES ($1, $2)`,
		nodeID,
		string(hash),
	)

	return err
}

func (r *Repository) ValidateAPIKey(
	ctx context.Context,
	nodeID string,
	apiKey string,
) bool {
	var hash string

	err := r.DB.QueryRowContext(
		ctx,
		`SELECT api_key_hash
		 FROM nodes
		 WHERE node_id = $1`,
		nodeID,
	).Scan(&hash)

	if err != nil {
		return false
	}

	return bcrypt.CompareHashAndPassword(
		[]byte(hash),
		[]byte(apiKey),
	) == nil
}