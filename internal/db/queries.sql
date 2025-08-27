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
