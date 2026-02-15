CREATE TABLE IF NOT EXISTS session_operators (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  session_id UUID NOT NULL REFERENCES streaming_sessions(id) ON DELETE CASCADE,
  user_id UUID NOT NULL,
  connected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_session_operators_session_id ON session_operators(session_id);
CREATE INDEX IF NOT EXISTS idx_session_operators_user_id ON session_operators(user_id);
