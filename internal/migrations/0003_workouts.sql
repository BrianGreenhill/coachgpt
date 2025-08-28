-- +goose Up
CREATE TABLE IF NOT EXISTS workout (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  athlete_id    UUID NOT NULL REFERENCES athlete(id) ON DELETE CASCADE,
  source TEXT NOT NULL DEFAULT 'strava',
  source_id    BIGINT NOT NULL,                     -- e.g. Strava activity ID
  name TEXT,
  sport TEXT NOT NULL,
  started_at TIMESTAMPTZ NOT NULL,
  duration_sec INT NOT NULL,
  distance_m FLOAT,
  elev_gain_m FLOAT,
  avg_hr INT,
  raw_json JSONB NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (athlete_id, source, source_id)
);

ALTER TABLE athlete
  ADD COLUMN IF NOT EXISTS last_strava_sync TIMESTAMPTZ;

-- +goose Down
DROP TABLE IF EXISTS workout;
ALTER TABLE athlete DROP COLUMN IF EXISTS last_strava_sync;
