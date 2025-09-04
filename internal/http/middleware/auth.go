package middleware

import "net/http"

type contextKey string

const CoachIDKey contextKey = "coach_id"

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		coachID := r.Context().Value(CoachIDKey)
		if coachID == nil || coachID == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
