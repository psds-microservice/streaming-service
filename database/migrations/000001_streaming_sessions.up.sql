CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS streaming_sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id UUID NOT NULL,
  stream_key VARCHAR(64) NOT NULL UNIQUE,
  status VARCHAR(20) NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting', 'active', 'finished')),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  finished_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX IF NOT EXISTS idx_streaming_sessions_client_id ON streaming_sessions(client_id);
CREATE INDEX IF NOT EXISTS idx_streaming_sessions_status ON streaming_sessions(status);

CREATE OR REPLACE FUNCTION update_streaming_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_update_streaming_sessions_updated_at ON streaming_sessions;
CREATE TRIGGER trigger_update_streaming_sessions_updated_at
  BEFORE UPDATE ON streaming_sessions
  FOR EACH ROW
  EXECUTE PROCEDURE update_streaming_sessions_updated_at();
