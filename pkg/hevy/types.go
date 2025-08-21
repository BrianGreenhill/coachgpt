package hevy

// Matches your sample JSON exactly (nullable fields as pointers)
type Body struct {
	Page      int           `json:"page"`
	PageCount int           `json:"page_count"`
	Workouts  []WorkoutJSON `json:"workouts"`
}

type WorkoutJSON struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	StartTime   string         `json:"start_time"`
	EndTime     string         `json:"end_time"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	Exercises   []ExerciseJSON `json:"exercises"`
}

type ExerciseJSON struct {
	Index              int       `json:"index"`
	Title              string    `json:"title"`
	Notes              string    `json:"notes"`
	ExerciseTemplateID string    `json:"exercise_template_id"`
	SupersetID         *string   `json:"superset_id"`
	Sets               []SetJSON `json:"sets"`
}

type SetJSON struct {
	Index           int      `json:"index"`
	Type            string   `json:"type"` // "normal","dropset","failure",...
	WeightKG        *float64 `json:"weight_kg"`
	Reps            *int     `json:"reps"`
	DistanceMeters  *float64 `json:"distance_meters"`
	DurationSeconds *int     `json:"duration_seconds"`
	RPE             *float64 `json:"rpe"`
	CustomMetric    any      `json:"custom_metric"`
}
