package storage

import (
	"context"
	"testing"

	"github.com/m4rba4s/Nexus-LLM/internal/kinematics"
)

// TestEphemeralMode_NilPool verifies that all Storage methods gracefully
// no-op when pool is nil (DATABASE_URL not set).
func TestEphemeralMode_NilPool(t *testing.T) {
	t.Parallel()

	s := &Storage{pool: nil}
	ctx := context.Background()

	t.Run("SaveMemory returns nil", func(t *testing.T) {
		t.Parallel()
		err := s.SaveMemory(ctx, "user1", "user", "hello", []float32{0.1, 0.2, 0.3})
		if err != nil {
			t.Errorf("SaveMemory with nil pool: got %v, want nil", err)
		}
	})

	t.Run("SearchSimilarMemories returns nil", func(t *testing.T) {
		t.Parallel()
		memories, err := s.SearchSimilarMemories(ctx, "user1", []float32{0.1, 0.2}, 5)
		if err != nil {
			t.Errorf("SearchSimilarMemories with nil pool: got %v, want nil", err)
		}
		if memories != nil {
			t.Errorf("expected nil memories, got %v", memories)
		}
	})

	t.Run("Close does not panic", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Close panicked with nil pool: %v", r)
			}
		}()
		s.Close()
	})
}

func TestNewStorage_NoDatabaseURL(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv
	t.Setenv("DATABASE_URL", "")

	store, err := NewStorage(context.Background())
	if err != nil {
		t.Fatalf("NewStorage with no DATABASE_URL: unexpected error %v", err)
	}
	if store != nil {
		t.Fatal("NewStorage with no DATABASE_URL: expected nil Storage, got non-nil")
	}
}

func TestSemanticMemory_ZeroValue(t *testing.T) {
	t.Parallel()

	var mem SemanticMemory
	if mem.ID != 0 {
		t.Errorf("zero-value ID = %d, want 0", mem.ID)
	}
	if mem.Role != "" {
		t.Errorf("zero-value Role = %q, want empty", mem.Role)
	}
	if mem.Content != "" {
		t.Errorf("zero-value Content = %q, want empty", mem.Content)
	}
	if mem.Embedding != nil {
		t.Errorf("zero-value Embedding should be nil")
	}
	if mem.Distance != 0.0 {
		t.Errorf("zero-value Distance = %f, want 0.0", mem.Distance)
	}
}

// TestStorage_MethodsRequirePool verifies that methods needing a pool
// correctly panic or return errors when called on a non-nil Storage with nil pool.
func TestStorage_MethodsRequirePool(t *testing.T) {
	t.Parallel()

	s := &Storage{pool: nil}
	ctx := context.Background()

	t.Run("InitSchema panics with nil pool", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("InitSchema with nil pool should panic (nil pointer dereference)")
			}
		}()
		_ = s.InitSchema(ctx)
	})

	t.Run("GetOrCreateUser panics with nil pool", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("GetOrCreateUser with nil pool should panic (nil pointer dereference)")
			}
		}()
		_, _ = s.GetOrCreateUser(ctx, "test", kinematics.RoleUser)
	})

	t.Run("UpdateUserState panics with nil pool", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("UpdateUserState with nil pool should panic (nil pointer dereference)")
			}
		}()
		_ = s.UpdateUserState(ctx, kinematics.UserIdentity{ID: "test"})
	})

	t.Run("SaveSessionState panics with nil pool", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("SaveSessionState with nil pool should panic (nil pointer dereference)")
			}
		}()
		_ = s.SaveSessionState(ctx, "test", kinematics.StateVector{})
	})
}
