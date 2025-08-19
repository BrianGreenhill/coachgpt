package main

import (
	"briangreenhill/coachgpt/hevy"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

const (
	stravaAuthBase  = "https://www.strava.com/oauth/authorize"
	stravaTokenURL  = "https://www.strava.com/oauth/token"
	stravaApiBase   = "https://www.strava.com/api/v3"
	stravaTokenFile = "strava_token.json"
	stravaRedirect  = "http://127.0.0.1:8723/cb"
)

var noCache = os.Getenv("STRAVA_NOCACHE") == "1"

type Tokens struct {
	TokenType    string `json:"token_type"`
	AccessToken  string `json:"access_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

func homeFile(name string) (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, name), nil
}

func loadTokens() (*Tokens, error) {
	path, err := homeFile(stravaTokenFile)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Tokens
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func saveTokens(t *Tokens) error {
	path, err := homeFile(stravaTokenFile)
	if err != nil {
		return err
	}
	b, _ := json.MarshalIndent(t, "", "  ")
	return os.WriteFile(path, b, 0600)
}

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("environment variable %s is not set", key)
	}
	return val
}

func ensureTokens(clientID, clientSecret string) string {
	// 1) Try to load tokens
	tok, _ := loadTokens()
	now := time.Now().Unix()

	// 2) If we have tokens, refresh if needed; otherwise return
	if tok != nil && tok.RefreshToken != "" {
		// If expiring in < 2 min, refresh
		if tok.ExpiresAt-now < 120 {
			form := url.Values{
				"client_id":     {clientID},
				"client_secret": {clientSecret},
				"grant_type":    {"refresh_token"},
				"refresh_token": {tok.RefreshToken},
			}
			resp, err := http.PostForm(stravaTokenURL, form)
			if err != nil {
				log.Fatalf("refresh token failed: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Fatalf("refresh token failed: %s", resp.Status)
			}
			var nt Tokens
			if err := json.NewDecoder(resp.Body).Decode(&nt); err != nil {
				log.Fatalf("decode refresh token failed: %v", err)
			}
			// update and persist
			*tok = nt
			if err := saveTokens(tok); err != nil {
				log.Fatalf("save token failed: %v", err)
			}
		}
		// Either refreshed or still valid
		return tok.AccessToken
	}

	// 3) No tokens yet → perform OAuth on localhost:8723
	type result struct {
		code string
		err  error
	}
	resCh := make(chan result, 1)

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    "127.0.0.1:8723",
		Handler: mux,
	}

	mux.HandleFunc("/cb", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := q.Get("code")
		if code == "" {
			http.Error(w, "no code in query", http.StatusBadRequest)
			resCh <- result{"", fmt.Errorf("no code")}
			return
		}
		fmt.Fprintln(w, "Authorized. You can close this window.")
		// return code to main goroutine then shut server down
		resCh <- result{code: code}
		go func() { _ = srv.Shutdown(context.Background()) }()
	})

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		log.Fatalf("listen %s: %v", srv.Addr, err)
	}
	go func() {
		// Serve returns http.ErrServerClosed on Shutdown — that’s fine
		_ = srv.Serve(ln)
	}()

	// Build the authorize URL
	authURL := fmt.Sprintf(
		"%s?client_id=%s&response_type=code&redirect_uri=%s&scope=%s&approval_prompt=auto",
		stravaAuthBase,
		url.QueryEscape(clientID),
		url.QueryEscape(stravaRedirect),
		url.QueryEscape("read,activity:read_all"),
	)
	fmt.Println("Open in browser:", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Println("If the browser didn’t open automatically, copy/paste the URL above.")
	}

	// Wait for callback
	res := <-resCh
	if res.err != nil || res.code == "" {
		log.Fatalf("OAuth failed: %v", res.err)
	}

	// Exchange code for tokens
	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {res.code},
		"grant_type":    {"authorization_code"},
	}
	resp, err := http.PostForm(stravaTokenURL, form)
	if err != nil {
		log.Fatalf("token exchange failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("token exchange failed: %s", resp.Status)
	}
	var nt Tokens
	if err := json.NewDecoder(resp.Body).Decode(&nt); err != nil {
		log.Fatalf("decode tokens failed: %v", err)
	}
	if err := saveTokens(&nt); err != nil {
		log.Fatalf("save token failed: %v", err)
	}
	return nt.AccessToken

}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", u).Start()
	case "windows":
		// this is the most reliable on modern Windows
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

type Split struct {
	Split               int     `json:"split"`                // 1,2,3...
	Distance            float64 `json:"distance"`             // meters (usually ~1000)
	ElapsedTime         int     `json:"elapsed_time"`         // sec
	MovingTime          int64   `json:"moving_time"`          // sec
	AverageSpeed        float64 `json:"average_speed"`        // m/s
	ElevationDifference float64 `json:"elevation_difference"` // meters
	PaceZone            int     `json:"pace_zone"`
}

type Lap struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`              // e.g., "Lap 1"
	LapIndex         int     `json:"lap_index"`         // 1-based
	Split            int     `json:"split"`             // sometimes present
	ElapsedTime      int     `json:"elapsed_time"`      // sec
	MovingTime       int64   `json:"moving_time"`       // sec
	Distance         float64 `json:"distance"`          // meters
	AverageSpeed     float64 `json:"average_speed"`     // m/s
	MaxSpeed         float64 `json:"max_speed"`         // m/s
	AverageHeartrate float64 `json:"average_heartrate"` // bpm (may be 0)
	MaxHeartrate     float64 `json:"max_heartrate"`     // bpm
	ElevationGain    float64 `json:"total_elevation_gain"`
	ElevationLoss    float64 `json:"total_elevation_loss"`
	StartIndex       int     `json:"start_index"` // stream index (optional)
	EndIndex         int     `json:"end_index"`   // stream index (optional)
}

type Activity struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	SportType          string  `json:"sport_type"`
	MovingTime         int64   `json:"moving_time"`
	Distance           float64 `json:"distance"`
	AverageSpeed       float64 `json:"average_speed"`
	AverageHeartRate   float64 `json:"average_heartrate"`
	TotalElevationGain float64 `json:"total_elevation_gain"`
	StartDateLocal     string  `json:"start_date_local"` // e.g. "2023-10-01T10:00:00Z"

	SplitsMetric []Split `json:"splits_metric"` // metric splits, may be empty
}

func stravaApiGETCached(token, path string, params map[string]string, out any, ttl time.Duration) error {
	u := stravaApiBase + path
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if params != nil {
		q := req.URL.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	cacheName := keyFor(path, params)
	// Try fresh-enough cache
	if !noCache {
		if ce, err := readCache(cacheName, ttl); err == nil && len(ce.Body) > 0 {
			return json.Unmarshal(ce.Body, out)
		}
	}

	// Try ETag (revalidation) if we have any cache (even if stale)
	if ce, err := readCache(cacheName, 0); err == nil && ce.ETag != "" {
		req.Header.Set("If-None-Match", ce.ETag)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp.StatusCode == http.StatusNotModified {
			// 304 -> use cached body
			_ = resp.Body.Close()
			return json.Unmarshal(ce.Body, out)
		}
		// fall through to normal fetch if error or not 304
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	}

	// Normal fetch
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET %s -> %s: %s", path, resp.Status, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Save cache with new ETag (if any)
	ce := &cacheEntry{
		ETag: resp.Header.Get("ETag"),
		Body: json.RawMessage(body),
	}
	_ = writeCache(cacheName, ce)

	return json.Unmarshal(body, out)
}

type Stream struct {
	Data []float64 `json:"data"`
}

type Streams struct {
	Time           Stream `json:"time"`
	Heartrate      Stream `json:"heartrate"`
	VelocitySmooth Stream `json:"velocity_smooth"`
	Distance       Stream `json:"distance"`
	Altitude       Stream `json:"altitude"`
}

func getLatestRun(token string) (*Activity, error) {
	var activities []Activity
	if err := stravaApiGETCached(token, "/athlete/activities", map[string]string{"per_page": "10", "include_all_efforts": "true"}, &activities, 24*time.Hour); err != nil {
		return nil, err
	}
	for _, a := range activities {
		if a.SportType == "Run" || a.SportType == "TrailRun" {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("no recent run activity found")
}

func getActivity(token string, id int64) (*Activity, error) {
	var activity Activity
	err := stravaApiGETCached(token, fmt.Sprintf("/activities/%d", id), map[string]string{"include_all_efforts": "true"}, &activity, 24*time.Hour)
	return &activity, err
}

func getLaps(token string, id int64) ([]Lap, error) {
	var laps []Lap
	err := stravaApiGETCached(token, fmt.Sprintf("/activities/%d/laps", id), nil, &laps, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return laps, nil
}

func getStreams(token string, id int64) (*Streams, error) {
	var raw map[string]Stream
	err := stravaApiGETCached(token, fmt.Sprintf("/activities/%d/streams", id),
		map[string]string{"keys": "time,heartrate,velocity_smooth,distance,altitude", "key_by_type": "true"}, &raw, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	s := &Streams{}
	if v, ok := raw["time"]; ok {
		s.Time = v
	}
	if v, ok := raw["heartrate"]; ok {
		s.Heartrate = v
	}
	if v, ok := raw["velocity_smooth"]; ok {
		s.VelocitySmooth = v
	}
	if v, ok := raw["distance"]; ok {
		s.Distance = v
	}
	if v, ok := raw["altitude"]; ok {
		s.Altitude = v
	}
	return s, nil
}

func secToHHMM(sec int64) string {
	m := sec / 60
	h := m / 60
	m = m % 60
	return fmt.Sprintf("%d:%02d", h, m)
}

func paceFromMoving(distanceMeters float64, movingSec int64) string {
	if distanceMeters <= 0 || movingSec <= 0 {
		return "-"
	}
	secPerKm := float64(movingSec) / (distanceMeters / 1000.0) // sec/km
	m := int(secPerKm) / 60
	s := int(secPerKm) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func computeZones(hr []float64, hrmax int) (z [5]int) {
	// Z1 <70%, Z2 70-80%, Z3 80-88%, Z4 88-95%, Z5 >95%
	cuts := []float64{0.70, 0.80, 0.88, 0.95}
	for _, v := range hr {
		if v <= 0 {
			continue
		}
		r := v / float64(hrmax)
		switch {
		case r < cuts[0]:
			z[0]++
		case r < cuts[1]:
			z[1]++
		case r < cuts[2]:
			z[2]++
		case r < cuts[3]:
			z[3]++
		default:
			z[4]++
		}
	}
	return
}

type SplitHR struct {
	KM        int
	Pace      string
	AvgHR     int
	MaxHR     int
	ElevDelta int
}

func computeSplitHR(splits []Split, timeStream, hrStream []float64) []SplitHR {
	out := make([]SplitHR, 0, len(splits))
	if len(timeStream) == 0 || len(hrStream) == 0 {
		return out
	}
	if len(timeStream) != len(hrStream) {
		// zip to shorter
		if len(timeStream) > len(hrStream) {
			timeStream = timeStream[:len(hrStream)]
		} else {
			hrStream = hrStream[:len(timeStream)]
		}
	}

	elapsedSoFar := 0
	for i, sp := range splits {
		start := elapsedSoFar
		end := elapsedSoFar + sp.ElapsedTime
		elapsedSoFar = end

		sum, cnt, max := 0, 0, 0
		for idx, tF := range timeStream {
			t := int(tF)
			if t >= start && t < end {
				hr := int(hrStream[idx])
				if hr <= 0 {
					continue
				}
				sum += hr
				cnt++
				if hr > max {
					max = hr
				}
			}
		}
		avg := 0
		if cnt > 0 {
			avg = sum / cnt
		}

		out = append(out, SplitHR{
			KM:        i + 1,
			Pace:      paceFromMoving(sp.Distance, sp.MovingTime),
			AvgHR:     avg,
			MaxHR:     max,
			ElevDelta: int(math.Round(sp.ElevationDifference)),
		})
	}
	return out
}

func printSplitsWithHR(act *Activity, streams *Streams) {
	fmt.Println("KM | Pace | Avg HR | Max HR | Elevation")
	fmt.Println("---|------|--------|--------|---------")

	// If we have streams, compute true per-split HR:
	if streams != nil && len(streams.Time.Data) > 0 && len(streams.Heartrate.Data) > 0 {
		rows := computeSplitHR(act.SplitsMetric, streams.Time.Data, streams.Heartrate.Data)
		if len(rows) > 0 {
			for _, r := range rows {
				avg := "—"
				if r.AvgHR > 0 {
					avg = fmt.Sprintf("%d", r.AvgHR)
				}
				max := "—"
				if r.MaxHR > 0 {
					max = fmt.Sprintf("%d", r.MaxHR)
				}
				fmt.Printf("%d | %s | %s | %s | %+d m\n",
					r.KM, r.Pace, avg, max, r.ElevDelta)
			}
			return
		}
	}

	// Fallback: print pace/elev from splits_metric without HR
	for _, sp := range act.SplitsMetric {
		fmt.Printf("%d | %s | %s | %s | %+d m\n",
			sp.Split,
			paceFromMoving(sp.Distance, sp.MovingTime),
			"—", "—",
			int(math.Round(sp.ElevationDifference)),
		)
	}
}

type LapElev struct{ Gain, Loss, Net int }

func lapElevationFromStreams(l Lap, altitude []float64) LapElev {
	// Guard indices
	start := l.StartIndex
	end := l.EndIndex
	if start < 0 {
		start = 0
	}
	if end > len(altitude) {
		end = len(altitude)
	}
	if end <= start || len(altitude) == 0 {
		return LapElev{}
	}
	gain, loss := 0.0, 0.0
	prev := altitude[start]
	for i := start + 1; i < end; i++ {
		diff := altitude[i] - prev
		if diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
		prev = altitude[i]
	}
	net := gain - loss
	return LapElev{Gain: int(math.Round(gain)), Loss: int(math.Round(loss)), Net: int(math.Round(net))}
}

func printLapsWithElevation(laps []Lap, streams *Streams) {
	fmt.Println("Lap | Time | Dist | Pace | Avg HR | Max HR | Gain | Loss | Net")
	fmt.Println("---|------|------|------|--------|-------|------|------|-----")

	for _, lp := range laps {
		distKm := lp.Distance / 1000.0
		avgHR := "—"
		if lp.AverageHeartrate > 0 {
			avgHR = fmt.Sprintf("%d", int(lp.AverageHeartrate))
		}
		maxHR := "—"
		if lp.MaxHeartrate > 0 {
			maxHR = fmt.Sprintf("%d", int(lp.MaxHeartrate))
		}

		// Prefer API-provided gain/loss; else compute from altitude stream
		gain := int(math.Round(lp.ElevationGain))
		loss := int(math.Round(lp.ElevationLoss))
		net := gain - loss

		if (gain == 0 && loss == 0) && streams != nil && len(streams.Altitude.Data) > 0 &&
			lp.StartIndex >= 0 && lp.EndIndex > lp.StartIndex {
			elev := lapElevationFromStreams(lp, streams.Altitude.Data)
			gain, loss, net = elev.Gain, elev.Loss, elev.Net
		}

		fmt.Printf("%d | %d:%02d | %.2f km | %s | %s | %s | %+d m | %+d m | %+d m\n",
			lp.LapIndex,
			lp.MovingTime/60, lp.MovingTime%60,
			distKm,
			paceFromMoving(lp.Distance, lp.MovingTime),
			avgHR, maxHR,
			gain, loss, net,
		)
	}
}

func runCLI(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "help", "--help", "-h":
			fmt.Println("Usage: coachgpt [options]")
			fmt.Println("Options:")
			fmt.Println("  --help, -h          Show this help message")
			fmt.Println("  STRAVA_CLIENT_ID    Your Strava client ID (required)")
			fmt.Println("  STRAVA_CLIENT_SECRET Your Strava client secret (required)")
			fmt.Println("  STRAVA_HRMAX        Your maximum heart rate (required, e.g. 185)")
			fmt.Println("  STRAVA_ACTIVITY_ID  Specific activity ID to fetch (optional)")
			fmt.Println("  STRAVA_NOCACHE      Disable caching (optional)")
		case "version", "--version", "-v":
			fmt.Println("CoachGPT v0.1.0")
		case "strength", "--strength", "-s":
			runStrengthIntegration()
		default:
			return fmt.Errorf("unknown command: %s", args[0])
		}
	} else {
		runStravaIntegration()
	}

	return nil
}

func main() {
	if err := runCLI(os.Args[1:]); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func runStrengthIntegration() {
	apiKey := mustEnv("HEVY_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set HEVY_API_KEY environment variable")
	}

	cache, _ := hevy.NewFileCache("")
	client, err := hevy.New(apiKey, hevy.WithCache(cache, 24*time.Hour))
	if err != nil {
		log.Fatalf("Failed to create Hevy client: %v", err)
	}

	ctx := context.Background()

	w, err := client.GetLatestWorkout(ctx)
	if err != nil {
		log.Fatalf("Failed to get latest Hevy workout: %v", err)
	}

	fmt.Println("--- Paste below ---")
	fmt.Println("## Strength Log")
	fmt.Println("Title: " + w.Title)
	startTime, _ := time.Parse(time.RFC3339, w.StartTime)
	endTime, _ := time.Parse(time.RFC3339, w.EndTime)
	duration := endTime.Sub(startTime)
	fmt.Printf("Duration: %s\n", secToHHMM(int64(duration.Seconds())))
	totalVol := 0.0
	totalReps := 0

	fmt.Println("Exercises:")
	for _, ex := range w.Exercises {
		exVol := 0.0
		exReps := 0
		fmt.Printf("- %s\n", ex.Title)
		for _, s := range ex.Sets {
			switch {
			case s.Reps != nil && s.WeightKG != nil:
				exReps += *s.Reps
				totalReps += *s.Reps
				vol := float64(*s.Reps) * *s.WeightKG
				exVol += vol
				totalVol += vol
				fmt.Printf("  • Set %d: %d reps @ %.1f kg\n", s.Index+1, *s.Reps, *s.WeightKG)
			case s.DurationSeconds != nil:
				fmt.Printf("  • Set %d: %ds (time)\n", s.Index+1, *s.DurationSeconds)
			default:
				fmt.Printf("  • Set %d: (type=%s)\n", s.Index+1, s.Type)
			}
		}
	}

	fmt.Printf("Total Volume: %.1f kg\n", totalVol)
	fmt.Printf("Total Reps: %d\n", totalReps)

	fmt.Println("Notes: []")
	fmt.Println("RPE: 0-10 (0=rest, 10=max effort)")
	fmt.Println("Fueling: [pre + during]")
	fmt.Println("--- End paste ---")
}

func runStravaIntegration() {
	clientID := mustEnv("STRAVA_CLIENT_ID")
	clientSecret := mustEnv("STRAVA_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("Please set STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET environment variables")
	}
	hrmaxStr := os.Getenv("STRAVA_HRMAX")
	if hrmaxStr == "" {
		log.Fatal("Please set STRAVA_HRMAX environment variable")
	}
	hrmax, err := strconv.Atoi(hrmaxStr)
	if err != nil || hrmax < 120 {
		log.Fatalf("HR_MAX must be an int like 185-200")
	}
	var activityID int64
	if v := os.Getenv("STRAVA_ACTIVITY_ID"); v != "" {
		activityID, err = strconv.ParseInt(v, 10, 64)
		if err != nil || activityID <= 0 {
			log.Fatalf("STRAVA_ACTIVITY_ID must be a positive integer")
		}
	}
	token := ensureTokens(clientID, clientSecret)

	var act *Activity
	if activityID > 0 {
		act, err = getActivity(token, activityID)
	} else {
		latest, err2 := getLatestRun(token)
		if err2 != nil {
			log.Fatalf("failed to get latest run: %v", err2)
		}
		act, err = getActivity(token, latest.ID)
	}
	if err != nil {
		log.Fatalf("failed to get activity: %v", err)
	}

	streams, _ := getStreams(token, act.ID)
	var zones [5]int
	if streams != nil && len(streams.Heartrate.Data) > 0 {
		zones = computeZones(streams.Heartrate.Data, hrmax)
	}

	avgHR := "-"
	if act.AverageHeartRate > 0 {
		avgHR = fmt.Sprintf("%d", int(act.AverageHeartRate))
	}

	fmt.Println("--- Paste below ---")
	fmt.Println("## Run Log")
	typ := "Run"
	if act.SportType == "TrailRun" {
		typ = "Trail Run"
	}
	fmt.Printf("- **Type:** [%s] %s\n", typ, act.Name)
	when := act.StartDateLocal
	if when == "" {
		when = time.Now().Format(time.RFC3339)
	}
	fmt.Printf("- **When:** %s\n", when)
	fmt.Printf("- **Duration:** %s\n", secToHHMM(act.MovingTime))
	fmt.Printf("- **Distance:** %.1f (elev %d m)\n", act.Distance/1000.0, int(math.Round(act.TotalElevationGain)))
	fmt.Printf("- **Avg Pace:** %s / km\n", paceFromMoving(act.Distance, act.MovingTime))
	fmt.Printf("- **Avg HR:** %s bpm\n", avgHR)

	if streams != nil && len(streams.Heartrate.Data) > 0 {
		total := 0
		for _, v := range zones {
			total += v
		}
		if total == 0 {
			total = 1
		}
		toPct := func(n int) int { return int(math.Round(float64(n) / float64(total) * 100)) }
		fmt.Printf("- **HR Zones:** Z1: %d%%, Z2: %d%%, Z3: %d%%, Z4: %d%%, Z5: %d%%\n",
			toPct(zones[0]), toPct(zones[1]), toPct(zones[2]), toPct(zones[3]), toPct(zones[4]))
	} else {
		fmt.Println("- **HR Zones:** No heart rate data available")
	}

	fmt.Println("- **Splits:**")
	printSplitsWithHR(act, streams)

	laps, _ := getLaps(token, act.ID)
	fmt.Println("- **Laps:**")
	printLapsWithElevation(laps, streams)

	fmt.Println("- **RPE:** 0-10 (0=rest, 10=max effort)")
	fmt.Println("- **Fueling** [pre + during]")
	fmt.Println("- **Terrain/Weather:** []")
	fmt.Println("- **Notes:** []")
	fmt.Println("--- End paste ---")
}
