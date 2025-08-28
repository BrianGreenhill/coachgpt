package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/briangreenhill/coachgpt/internal/config"
	"github.com/briangreenhill/coachgpt/internal/db"
	"github.com/briangreenhill/coachgpt/internal/jobs"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("unable to connect to database:", err)
	}
	defer pool.Close()
	q := db.New(pool)

	srv := asynq.NewServer(asynq.RedisClientOpt{Addr: cfg.RedisAddr}, asynq.Config{
		Concurrency:    8,
		StrictPriority: false,
		Queues: map[string]int{
			"sync":    10, // higher priority
			"default": 5,  // default priority
		},
	})
	mux := asynq.NewServeMux()

	mux.HandleFunc(jobs.TaskSyncStrava, func(ctx context.Context, t *asynq.Task) error {
		var p jobs.SyncStravaPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			log.Printf("[asynq] bad payload: %v", err)
			return err
		}
		log.Printf("[sync] start athlete=%s", p.AthleteID)
		start := time.Now()
		err := syncStravaForAthlete(ctx, q, cfg.Strava.ClientID, cfg.Strava.ClientSecret, p)
		duration := time.Since(start)

		if err != nil {
			// Check if error is retryable
			if isRetryableError(err) {
				log.Printf("[sync] retryable error athlete=%s duration=%v: %v", p.AthleteID, duration, err)
				return err // allow retry
			}
			log.Printf("[sync] permanent error athlete=%s duration=%v: %v (dropping job)", p.AthleteID, duration, err)
			return nil // don't retry permanent failures
		}
		log.Printf("[sync] done athlete=%s duration=%v", p.AthleteID, duration)
		return nil
	})

	log.Println("Worker running...")
	log.Fatal(srv.Run(mux))
}

// isRetryableError determines if an error should trigger a job retry
func isRetryableError(err error) bool {
	errStr := strings.ToLower(err.Error())

	// Network/connectivity issues - should retry
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "dns") {
		return true
	}

	// Strava rate limiting - should retry later
	if strings.Contains(errStr, "429") ||
		strings.Contains(errStr, "rate limit") {
		return true
	}

	// Temporary server errors - should retry
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	// Token refresh failures might be temporary
	if strings.Contains(errStr, "refresh strava token") {
		return true
	}

	// Everything else (auth failures, bad data, etc.) - don't retry
	return false
}

type stravaActivity struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	StartDate   string   `json:"start_date"`
	ElapsedSecs int      `json:"elapsed_time"`
	DistanceM   float64  `json:"distance"`
	TotalElevM  float64  `json:"total_elevation_gain"`
	AvgHR       *float64 `json:"average_heartrate,omitempty"`
}

type tokenResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func syncStravaForAthlete(ctx context.Context, q *db.Queries, clientID, clientSecret string, p jobs.SyncStravaPayload) error {
	aid := uuid.MustParse(p.AthleteID)

	athlete, err := q.GetAthlete(ctx, aid)
	if err != nil {
		return fmt.Errorf("get athlete: %w", err)
	}

	access := athlete.StravaAccessToken.String
	refresh := athlete.StravaRefreshToken.String
	expiry := athlete.StravaTokenExpiry.Time

	if time.Until(expiry) < 2*time.Minute {
		tok, err := refreshStravaToken(clientID, clientSecret, refresh)
		if err != nil {
			return fmt.Errorf("refresh strava token: %w", err)
		}
		access = tok.AccessToken
		refresh = tok.RefreshToken
		expiry = time.Unix(tok.ExpiresAt, 0)

		if err := q.UpdateAthleteStravaTokens(ctx, db.UpdateAthleteStravaTokensParams{
			ID:                 aid,
			StravaAccessToken:  pgtype.Text{String: access, Valid: true},
			StravaRefreshToken: pgtype.Text{String: refresh, Valid: true},
			StravaTokenExpiry:  pgtype.Timestamptz{Time: expiry, Valid: true},
		}); err != nil {
			return fmt.Errorf("update athlete strava tokens: %w", err)
		}
	}

	since := time.Now().AddDate(0, 0, -14) // default 14 days
	if athlete.LastStravaSync.Valid {
		since = athlete.LastStravaSync.Time.Add(-12 * time.Hour) // back up 12 hours to be safe
	}
	if p.SinceUnix != 0 {
		since = time.Unix(p.SinceUnix, 0)
		log.Printf("[sync] athlete=%s using custom since time: %v", aid, since)
	}

	httpClient := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}

	after := strconv.FormatInt(since.Unix(), 10)
	page := 1
	total := 0

	for {
		url := fmt.Sprintf("https://www.strava.com/api/v3/athlete/activities?after=%s&per_page=50&page=%d", after, page)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+access)
		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("fetch strava activities: %w", err)
		}
		if resp.StatusCode == 401 {
			_ = resp.Body.Close()
			tok, err := refreshStravaToken(clientID, clientSecret, refresh)
			if err != nil {
				return fmt.Errorf("401/refresh: %w", err)
			}
			access, refresh, expiry = tok.AccessToken, tok.RefreshToken, time.Unix(tok.ExpiresAt, 0)
			if err := q.UpdateAthleteStravaTokens(ctx, db.UpdateAthleteStravaTokensParams{
				ID:                 aid,
				StravaAccessToken:  pgtype.Text{String: access, Valid: true},
				StravaRefreshToken: pgtype.Text{String: refresh, Valid: true},
				StravaTokenExpiry:  pgtype.Timestamptz{Time: expiry, Valid: true},
			}); err != nil {
				return fmt.Errorf("401/update tokens: %w", err)
			}
			continue // re-loop with new token
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode >= 300 {
			return fmt.Errorf("strava activities status %d: %s", resp.StatusCode, string(body))
		}

		var items []stravaActivity
		if err := json.Unmarshal(body, &items); err != nil {
			return fmt.Errorf("unmarshal strava activities: %w", err)
		}

		if len(items) == 0 {
			break
		}

		for _, a := range items {
			startedAt, _ := time.Parse(time.RFC3339, a.StartDate)
			avgHR := 0
			if a.AvgHR != nil {
				avgHR = int(*a.AvgHR)
			}

			// Upsert
			err := q.UpsertWorkout(ctx, db.UpsertWorkoutParams{
				AthleteID:   aid,
				SourceID:    a.ID,
				Name:        pgtype.Text{String: a.Name, Valid: a.Name != ""},
				Sport:       a.Type,
				StartedAt:   pgtype.Timestamptz{Time: startedAt, Valid: true},
				DurationSec: int32(a.ElapsedSecs),
				DistanceM:   pgtype.Float8{Float64: a.DistanceM, Valid: a.DistanceM > 0},
				ElevGainM:   pgtype.Float8{Float64: a.TotalElevM, Valid: a.TotalElevM > 0},
				AvgHr:       pgtype.Int4{Int32: int32(avgHR), Valid: avgHR > 0},
				RawJson:     bodySliceToJSONB(a),
			})
			if err != nil {
				return fmt.Errorf("upsert workout: %w", err)
			}
			total++
		}
		page++
	}

	if err := q.UpdateAthleteLastStravaSync(ctx, db.UpdateAthleteLastStravaSyncParams{
		ID:             aid,
		LastStravaSync: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}); err != nil {
		return fmt.Errorf("update athlete last strava sync: %w", err)
	}

	log.Printf("[sync] athlete=%s synced %d activities since %v", aid, total, since)
	return nil
}

func bodySliceToJSONB(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func refreshStravaToken(clientID, clientSecret, refreshToken string) (*tokenResp, error) {
	form := fmt.Sprintf("client_id=%s&client_secret=%s&grant_type=refresh_token&refresh_token=%s",
		clientID, clientSecret, refreshToken,
	)

	req, _ := http.NewRequest("POST", "https://www.strava.com/oauth/token", io.NopCloser(strings.NewReader(form)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("strava token status %d: %s", resp.StatusCode, string(b))
	}
	var tok tokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	return &tok, nil
}
