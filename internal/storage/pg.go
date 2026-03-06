package storage

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvector "github.com/pgvector/pgvector-go/pgx"
	"github.com/m4rba4s/Nexus-LLM/internal/kinematics"
)

// Storage handles PostgreSQL connections and queries for Persistent Memory
type Storage struct {
	pool *pgxpool.Pool
}

// NewStorage initializes a connection to PostgreSQL (Supabase or local).
// Expects DATABASE_URL in the environment.
// Returns nil if no DB URL is provided (Dry Run Mode).
func NewStorage(ctx context.Context) (*Storage, error) {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		fmt.Println("[STORAGE] DATABASE_URL not set. Running in Ephemeral Memory Mode.")
		return nil, nil // Return nil pointer to signify no storage
	}

	config, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DB config: %w", err)
	}

	// Minimal connection pool settings for CLI/Worker
	config.MaxConns = 10
	config.MinConns = 1
	config.MaxConnLifetime = time.Hour

	// Register vector type globally on connections
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvector.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database unreachable: %w", err)
	}

	// Register vector type for all connections
	// Note: We need to make sure the extension is created first.
	_, err = pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return nil, fmt.Errorf("failed to create vector extension: %w", err)
	}

	store := &Storage{pool: pool}
	if err := store.InitSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// InitSchema ensures the tables exist.
func (s *Storage) InitSchema(ctx context.Context) error {
	const schema = `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		messenger_id VARCHAR(255) UNIQUE NOT NULL,
		role VARCHAR(50) DEFAULT 'user',
		trust_score FLOAT DEFAULT 0.5,
		fatigue_level FLOAT DEFAULT 0.0,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_users_messenger_id ON users(messenger_id);

	CREATE TABLE IF NOT EXISTS cognitive_sessions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		session_summary TEXT,
		state_paranoia FLOAT DEFAULT 0.1,
		state_curiosity FLOAT DEFAULT 0.5,
		state_frustration FLOAT DEFAULT 0.0,
		is_active BOOLEAN DEFAULT TRUE,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS long_term_memory (
		id SERIAL PRIMARY KEY,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		role VARCHAR(50),
		content TEXT,
		embedding vector(768),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_memory_embedding ON long_term_memory USING hnsw (embedding vector_cosine_ops);
	`
	_, err := s.pool.Exec(ctx, schema)
	return err
}

// GetOrCreateUser fetches a user by their messenger/cli ID, creating one if it doesn't exist.
func (s *Storage) GetOrCreateUser(ctx context.Context, messengerID string, defaultRole kinematics.UserRole) (kinematics.UserIdentity, error) {
	var user kinematics.UserIdentity
	var uuidStr string

	err := s.pool.QueryRow(ctx, `
		SELECT id, messenger_id, role, trust_score, fatigue_level
		FROM users WHERE messenger_id = $1
	`, messengerID).Scan(&uuidStr, &user.ID, &user.Role, &user.TrustScore, &user.Fatigue)

	if err != nil {
		// If not found, create new
		if err.Error() == "no rows in result set" {
			err = s.pool.QueryRow(ctx, `
				INSERT INTO users (messenger_id, role)
				VALUES ($1, $2)
				RETURNING id, messenger_id, role, trust_score, fatigue_level
			`, messengerID, defaultRole).Scan(&uuidStr, &user.ID, &user.Role, &user.TrustScore, &user.Fatigue)
			if err != nil {
				return user, fmt.Errorf("failed to create user: %w", err)
			}
		} else {
			return user, fmt.Errorf("failed to query user: %w", err)
		}
	} else {
		// Update last_seen
		_, _ = s.pool.Exec(ctx, `UPDATE users SET last_seen = CURRENT_TIMESTAMP WHERE id = $1`, uuidStr)
	}

	// Ensure ID contains the messenger_id for our App Logic
	user.ID = messengerID

	return user, nil
}

// UpdateUserState updates the TrustScore and Fatigue in the DB.
func (s *Storage) UpdateUserState(ctx context.Context, ident kinematics.UserIdentity) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE users 
		SET trust_score = $1, fatigue_level = $2, last_seen = CURRENT_TIMESTAMP
		WHERE messenger_id = $3
	`, ident.TrustScore, ident.Fatigue, ident.ID)
	return err
}

// SaveSessionState saves the current kinematics vector for a given session.
func (s *Storage) SaveSessionState(ctx context.Context, messengerID string, state kinematics.StateVector) error {
	// Simple implementation: UPSERT the active cognitive session for the user
	// Real-world implementation would bind to session IDs
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cognitive_sessions (user_id, state_paranoia, state_curiosity, state_frustration, is_active)
		SELECT id, $1, $2, $3, TRUE
		FROM users WHERE messenger_id = $4
	`, state.Paranoia, state.Curiosity, state.Frustration, messengerID)
	return err
}

// Close gracefully closes the DB pool
func (s *Storage) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}
