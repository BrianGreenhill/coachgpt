package jobs

const TaskSyncStrava = "sync:strava_athlete"

type SyncStravaPayload struct {
	AthleteID string `json:"athlete_id"`
	SinceUnix int64  `json:"since_unix,omitempty"`
}
