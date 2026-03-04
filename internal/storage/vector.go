package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/pgvector/pgvector-go"
	"github.com/yourusername/gollm/internal/kinematics"
)

// SemanticMemory represents a chunk of stored context with its embedding
type SemanticMemory struct {
	ID        int
	Role      string
	Content   string
	Embedding []float32
	Distance  float64
	CreatedAt time.Time
}

// SaveMemory stores a conversational chunk and its vector embedding
func (s *Storage) SaveMemory(ctx context.Context, messengerID string, role, content string, embedding []float32) error {
	if s.pool == nil {
		return nil // Ephemeral mode
	}

	// First fetch UUID from messenger_id
	user, err := s.GetOrCreateUser(ctx, messengerID, kinematics.RoleUser)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get the actual UUID (GetOrCreateUser returns messengerID in ID struct field for our app logic, so we re-fetch)
	var userUUID string
	err = s.pool.QueryRow(ctx, "SELECT id FROM users WHERE messenger_id = $1", user.ID).Scan(&userUUID)
	if err != nil {
		return fmt.Errorf("failed to resolve user UUID: %w", err)
	}

	// Insert into long_term_memory using pgvector
	vec := pgvector.NewVector(embedding)
	_, err = s.pool.Exec(ctx, `
		INSERT INTO long_term_memory (user_id, role, content, embedding)
		VALUES ($1, $2, $3, $4)
	`, userUUID, role, content, vec)

	return err
}

// SearchSimilarMemories performs a semantic search using Cosine Similarity (<=>)
func (s *Storage) SearchSimilarMemories(ctx context.Context, messengerID string, queryEmbedding []float32, topK int) ([]SemanticMemory, error) {
	if s.pool == nil {
		return nil, nil // Ephemeral mode
	}

	var userUUID string
	err := s.pool.QueryRow(ctx, "SELECT id FROM users WHERE messenger_id = $1", messengerID).Scan(&userUUID)
	if err != nil {
		if err.Error() == "no rows in result set" {
			// No history for user yet
			return nil, nil
		}
		return nil, fmt.Errorf("failed to resolve user UUID: %w", err)
	}

	vec := pgvector.NewVector(queryEmbedding)

	// Query top K matches using Cosine Similarity <=>
	rows, err := s.pool.Query(ctx, `
		SELECT id, role, content, embedding, created_at, embedding <=> $1 AS distance
		FROM long_term_memory
		WHERE user_id = $2
		ORDER BY distance ASC
		LIMIT $3
	`, vec, userUUID, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to run semantic search: %w", err)
	}
	defer rows.Close()

	var memories []SemanticMemory
	for rows.Next() {
		var mem SemanticMemory
		var v pgvector.Vector
		if err := rows.Scan(&mem.ID, &mem.Role, &mem.Content, &v, &mem.CreatedAt, &mem.Distance); err != nil {
			return nil, err
		}
		mem.Embedding = v.Slice()
		memories = append(memories, mem)
	}

	return memories, nil
}
