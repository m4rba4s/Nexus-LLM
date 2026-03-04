# Persistent Memory Schema (PostgreSQL)

Для симуляции "Долговременной Памяти" и "Уровня Доверия" (Trust Score), система  NexusLLM использует реляционную структуру.

## ER-Diagram (Mental Model)

### 1. `users` (Identity & Trust)
Хранит профили пользователей и их текущий уровень доверия к системе.

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    messenger_id VARCHAR(255) UNIQUE NOT NULL, -- Telegram/Signal ID
    role VARCHAR(50) DEFAULT 'user',           -- 'root' ("Papa"), 'user', 'guest'
    trust_score FLOAT DEFAULT 0.5,             -- [0.0, 1.0]. Root ignores this.
    fatigue_level FLOAT DEFAULT 0.0,           -- Compute usage penalty for this user
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- B-Tree index for ultra-fast L7 Routing lookups (< 5ms)
CREATE INDEX idx_users_messenger_id ON users(messenger_id);
```

### 2. `cognitive_sessions` (Context Contextualization)
Группирует запросы в логические сессии (чтобы машина "помнила" о чем идет речь, пока не сменится контекст).

```sql
CREATE TABLE cognitive_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_summary TEXT,                      -- LLM-generated distillation of context
    state_paranoia FLOAT DEFAULT 0.1,          -- Saved state snapshot P
    state_curiosity FLOAT DEFAULT 0.5,         -- Saved state snapshot C
    state_frustration FLOAT DEFAULT 0.0,       -- Saved state snapshot F
    is_active BOOLEAN DEFAULT TRUE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

### 3. `memory_embeddings` (Vector Retrieval - PGO)
В будущем: для pgvector. Хранит семантику важных решений.

```sql
CREATE TABLE memory_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID REFERENCES cognitive_sessions(id) ON DELETE CASCADE,
    prompt_text TEXT,
    response_text TEXT,
    -- embedding vector(1536), -- Uncomment when pgvector is active
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
```

## Security Invariants
1. `role = 'root'` ("Papa"): 
   - Обходит все метрики `fatigue_level`.
   - `trust_score` технически игнорируется (всегда $\infty$ в логике).
   - Всегда имеет доступ к `Claude_4_6_Opus`.
2. `trust_score < 0.3`:
   - Запрет вызова `Math_Verifier_Agent`.
   - Принудительный даунгрейд до дешевых моделей, независисмо от $H_{score}$ промпта.
