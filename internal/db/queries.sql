-- name: CreateCoach :one
INSERT INTO coach (email, name, tz)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCoachByEmail :one
SELECT * FROM coach WHERE email = $1 LIMIT 1;

-- name: UpsertCoachByEmail :one
INSERT INTO coach (email, name, tz)
VALUES ($1, $2, $3)
ON CONFLICT (email)
DO UPDATE SET name = COALESCE(EXCLUDED.name, coach.name)
RETURNING *;

-- name: CreateAthlete :one
INSERT INTO athlete (coach_id, name, email, tz)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SetAthleteStravaTokens :exec
UPDATE athlete
SET strava_athlete_id = $2,
    strava_access_token = $3,
    strava_refresh_token = $4,
    strava_token_expiry = $5
WHERE id = $1;

-- name: ListAthletesByCoach :many
SELECT * FROM athlete WHERE coach_id = $1 ORDER BY created_at DESC;

-- name: GetAthlete :one
SELECT * FROM athlete WHERE id = $1 LIMIT 1;

-- name: UpdateAthleteStravaTokens :exec
UPDATE athlete
SET strava_access_token = $2,
    strava_refresh_token = $3,
    strava_token_expiry = $4
WHERE id = $1;

-- name: UpdateAthleteLastStravaSync :exec
UPDATE athlete
SET last_strava_sync = $2
WHERE id = $1;

-- name: UpsertWorkout :exec
INSERT INTO workout (
    athlete_id, source, source_id, name, sport, started_at,
    duration_sec, distance_m, elev_gain_m, avg_hr, raw_json
) VALUES ($1, 'strava', $2, $3, $4, $5, $6, $7, $8, $9, $10)
    ON CONFLICT (athlete_id, source, source_id) DO UPDATE
SET name = $3, sport=$4, started_at=$5,
    duration_sec=$6, distance_m=$7, elev_gain_m=$8, avg_hr=$9,
    raw_json=$10, updated_at=now();
