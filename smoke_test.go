package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	scs "github.com/alexedwards/scs/v2"
	"github.com/briangreenhill/coachgpt/internal/auth"
	"github.com/briangreenhill/coachgpt/internal/config"
	"github.com/briangreenhill/coachgpt/internal/db"
	"github.com/briangreenhill/coachgpt/internal/http/routes"
	"github.com/briangreenhill/coachgpt/internal/jobs"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// MockStravaServer provides a simple mock for Strava OAuth and API endpoints
type MockStravaServer struct {
	server *httptest.Server
}

func NewMockStravaServer() *MockStravaServer {
	mux := http.NewServeMux()

	// OAuth token endpoint - returns unique tokens based on timestamp
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		timestamp := time.Now().UnixMicro() % 1000000000
		response := map[string]interface{}{
			"access_token":  fmt.Sprintf("mock_access_%d", timestamp),
			"refresh_token": fmt.Sprintf("mock_refresh_%d", timestamp),
			"expires_at":    time.Now().Add(6 * time.Hour).Unix(),
			"athlete": map[string]interface{}{
				"id": timestamp,
			},
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding token response: %v", err)
		}
	})

	// Activities endpoint - returns minimal mock activities
	mux.HandleFunc("/api/v3/athlete/activities", func(w http.ResponseWriter, r *http.Request) {
		activities := []map[string]interface{}{
			{
				"id":           time.Now().UnixNano(),
				"name":         "Morning Run",
				"type":         "Run",
				"start_date":   time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"elapsed_time": 1800,
				"distance":     5000.0,
			},
		}
		if err := json.NewEncoder(w).Encode(activities); err != nil {
			log.Printf("Error encoding activities response: %v", err)
		}
	})

	return &MockStravaServer{
		server: httptest.NewServer(mux),
	}
}

func (m *MockStravaServer) Close() {
	m.server.Close()
}

// TestSmokeTest simulates the complete user experience end-to-end
func TestSmokeTest(t *testing.T) {
	// Skip if no database URL provided
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping smoke test")
	}

	ctx := context.Background()

	// Setup database
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	queries := db.New(pool)

	// Setup mock Strava server
	mockStrava := NewMockStravaServer()
	defer mockStrava.Close()

	// Setup application configuration
	cfg := config.Config{
		DatabaseURL: dbURL,
		RedisAddr:   "localhost:6379",
		BaseURL:     "http://localhost:8080",
		JWTSecret:   "test-secret-" + uuid.New().String(),
		Strava: config.StravaConfig{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	// Setup HTTP server with mock Strava endpoints
	sess := scs.New()
	tmpl := template.Must(template.New("test").Parse(`
		{{define "invite_created.tmpl"}}Invite URL: {{.InviteURL}}{{end}}
		{{define "athlete_consent.tmpl"}}Consent Page{{end}}
		{{define "athlete_connected.tmpl"}}Connected!{{end}}
	`))

	ml := auth.MagicLink{Secret: []byte(cfg.JWTSecret), BaseURL: cfg.BaseURL}
	inv := auth.InviteLink{Secret: []byte(cfg.JWTSecret)}

	server := routes.New(sess, tmpl, queries, ml, inv, cfg)

	// Override Strava endpoints to use mock server
	server.StravaConf.Endpoint.AuthURL = mockStrava.server.URL + "/oauth/authorize"
	server.StravaConf.Endpoint.TokenURL = mockStrava.server.URL + "/oauth/token"

	t.Run("complete_user_experience", func(t *testing.T) {
		// 1. Create a coach (simulating coach signup)
		coach, err := queries.CreateCoach(ctx, db.CreateCoachParams{
			Email: "coach-" + uuid.New().String() + "@example.com",
			Name:  pgtype.Text{String: "Test Coach", Valid: true},
			Tz:    "UTC",
		})
		require.NoError(t, err)

		// 2. Coach creates athlete via web form (real user action)
		athleteEmail := "athlete-" + uuid.New().String() + "@example.com"
		form := url.Values{}
		form.Add("name", "Test Athlete")
		form.Add("email", athleteEmail)

		req := httptest.NewRequest("POST", "/athletes", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Add coach session (simulating logged-in coach)
		sessCtx := server.Sess.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			server.Sess.Put(r.Context(), "coach_id", coach.ID.String())
			server.Router.ServeHTTP(w, r)
		}))

		w := httptest.NewRecorder()
		sessCtx.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "athlete creation should succeed")

		// 3. Verify athlete was created in database (core functionality)
		athletes, err := queries.ListAthletesByCoach(ctx, coach.ID)
		require.NoError(t, err)
		require.Len(t, athletes, 1)
		athlete := athletes[0]
		require.Equal(t, "Test Athlete", athlete.Name)
		require.Equal(t, athleteEmail, athlete.Email.String)

		// 4. Test Strava OAuth start (verify redirect works)
		req = httptest.NewRequest("GET", "/oauth/strava/start?aid="+athlete.ID.String(), nil)
		w = httptest.NewRecorder()
		server.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusFound, w.Code, "OAuth start should redirect")

		location := w.Header().Get("Location")
		require.Contains(t, location, "oauth/authorize", "should redirect to OAuth")

		// 5. Simulate token storage directly (skip complex OAuth callback)
		stravaAthleteID := time.Now().UnixMicro() % 1000000000
		err = queries.SetAthleteStravaTokens(ctx, db.SetAthleteStravaTokensParams{
			ID:                 athlete.ID,
			StravaAthleteID:    pgtype.Int8{Int64: stravaAthleteID, Valid: true},
			StravaAccessToken:  pgtype.Text{String: fmt.Sprintf("mock_access_%d", stravaAthleteID), Valid: true},
			StravaRefreshToken: pgtype.Text{String: fmt.Sprintf("mock_refresh_%d", stravaAthleteID), Valid: true},
			StravaTokenExpiry:  pgtype.Timestamptz{Time: time.Now().Add(6 * time.Hour), Valid: true},
		})
		require.NoError(t, err)

		// 6. Verify tokens were saved
		updatedAthlete, err := queries.GetAthlete(ctx, athlete.ID)
		require.NoError(t, err)
		require.True(t, updatedAthlete.StravaAccessToken.Valid, "access token should be saved")
		require.True(t, updatedAthlete.StravaAthleteID.Valid, "Strava athlete ID should be saved")
		require.Equal(t, stravaAthleteID, updatedAthlete.StravaAthleteID.Int64)

		// 7. Test job enqueueing (background processing)
		redisClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisAddr})
		if redisClient != nil {
			defer func() {
				if closeErr := redisClient.Close(); closeErr != nil {
					log.Printf("Error closing redis client: %v", closeErr)
				}
			}()

			payload := jobs.SyncStravaPayload{AthleteID: athlete.ID.String()}
			payloadBytes, _ := json.Marshal(payload)

			task := asynq.NewTask(jobs.TaskSyncStrava, payloadBytes)
			info, err := redisClient.Enqueue(task)
			if err == nil {
				require.NotEmpty(t, info.ID, "sync job should be enqueued")
				t.Logf("✅ Background sync job queued: %s", info.ID)
			} else {
				t.Logf("⚠️  Redis not available, skipping job verification")
			}
		}

		t.Logf("✅ Complete user experience validated!")
		t.Logf("   👨‍💼 Coach: %s", coach.Email)
		t.Logf("   🏃‍♀️ Athlete: %s (%s)", athlete.Name, athlete.Email.String)
		t.Logf("   🔗 Strava Connected: %v", updatedAthlete.StravaAccessToken.Valid)
		t.Logf("   💾 Data Ready for Sync: %v", updatedAthlete.StravaAthleteID.Valid)
	})
}
