package strava

import (
	"fmt"
	"math"
)

// PrintSplitsWithHR prints a formatted table of splits with heart rate data
func PrintSplitsWithHR(act *Activity, streams *Streams) {
	fmt.Println("KM | Pace | Avg HR | Max HR | Elevation")
	fmt.Println("---|------|--------|--------|---------")

	// If we have streams, compute true per-split HR:
	if streams != nil && len(streams.Time.Data) > 0 && len(streams.Heartrate.Data) > 0 {
		rows := ComputeSplitHR(act.SplitsMetric, streams.Time.Data, streams.Heartrate.Data)
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
			PaceFromMoving(sp.Distance, sp.MovingTime),
			"—", "—",
			int(math.Round(sp.ElevationDifference)),
		)
	}
}

// PrintLapsWithElevation prints a formatted table of laps with elevation data
func PrintLapsWithElevation(laps []Lap, streams *Streams) {
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
			elev := LapElevationFromStreams(lp, streams.Altitude.Data)
			gain, loss, net = elev.Gain, elev.Loss, elev.Net
		}

		fmt.Printf("%d | %d:%02d | %.2f km | %s | %s | %s | %+d m | %+d m | %+d m\n",
			lp.LapIndex,
			lp.MovingTime/60, lp.MovingTime%60,
			distKm,
			PaceFromMoving(lp.Distance, lp.MovingTime),
			avgHR, maxHR,
			gain, loss, net,
		)
	}
}

// FormatRunAnalysis generates the complete formatted analysis for an activity
func FormatRunAnalysis(act *Activity, streams *Streams, laps []Lap, hrmax int) string {
	var zones [5]int
	if streams != nil && len(streams.Heartrate.Data) > 0 {
		zones = ComputeZones(streams.Heartrate.Data, hrmax)
	}

	avgHR := "-"
	if act.AverageHeartRate > 0 {
		avgHR = fmt.Sprintf("%d", int(act.AverageHeartRate))
	}

	result := "--- Paste below ---\n"
	result += "## Run Log\n"

	typ := "Run"
	if act.SportType == "TrailRun" {
		typ = "Trail Run"
	}
	result += fmt.Sprintf("- **Type:** [%s] %s\n", typ, act.Name)

	when := act.StartDateLocal
	if when == "" {
		when = "Unknown"
	}
	result += fmt.Sprintf("- **When:** %s\n", when)
	result += fmt.Sprintf("- **Duration:** %s\n", SecToHHMM(act.MovingTime))
	result += fmt.Sprintf("- **Distance:** %.1f km (elev %d m)\n", act.Distance/1000.0, int(math.Round(act.TotalElevationGain)))
	result += fmt.Sprintf("- **Avg Pace:** %s / km\n", PaceFromMoving(act.Distance, act.MovingTime))
	result += fmt.Sprintf("- **Avg HR:** %s bpm\n", avgHR)

	if streams != nil && len(streams.Heartrate.Data) > 0 {
		total := 0
		for _, v := range zones {
			total += v
		}
		if total == 0 {
			total = 1
		}
		toPct := func(n int) int { return int(math.Round(float64(n) / float64(total) * 100)) }
		result += fmt.Sprintf("- **HR Zones:** Z1: %d%%, Z2: %d%%, Z3: %d%%, Z4: %d%%, Z5: %d%%\n",
			toPct(zones[0]), toPct(zones[1]), toPct(zones[2]), toPct(zones[3]), toPct(zones[4]))
	} else {
		result += "- **HR Zones:** No heart rate data available\n"
	}

	result += "- **Splits:**\n"
	// Note: For complete formatting, you might want to capture the print output
	// For now, this returns the basic structure

	result += "- **Laps:**\n"
	// Note: For complete formatting, you might want to capture the print output
	// For now, this returns the basic structure

	result += "- **RPE:** 0-10 (0=rest, 10=max effort)\n"
	result += "- **Fueling** [pre + during]\n"
	result += "- **Terrain/Weather:** []\n"
	result += "- **Notes:** []\n"
	result += "--- End paste ---\n"

	return result
}

// FormatSplitsWithHR returns a formatted string of splits with heart rate data
func FormatSplitsWithHR(act *Activity, streams *Streams) string {
	result := "KM | Pace | Avg HR | Max HR | Elevation\n"
	result += "---|------|--------|--------|---------\n"

	// If we have streams, compute true per-split HR:
	if streams != nil && len(streams.Time.Data) > 0 && len(streams.Heartrate.Data) > 0 {
		rows := ComputeSplitHR(act.SplitsMetric, streams.Time.Data, streams.Heartrate.Data)
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
				result += fmt.Sprintf("%d | %s | %s | %s | %+d m\n",
					r.KM, r.Pace, avg, max, r.ElevDelta)
			}
			return result
		}
	}

	// Fallback: print pace/elev from splits_metric without HR
	for _, sp := range act.SplitsMetric {
		result += fmt.Sprintf("%d | %s | %s | %s | %+d m\n",
			sp.Split,
			PaceFromMoving(sp.Distance, sp.MovingTime),
			"—", "—",
			int(math.Round(sp.ElevationDifference)),
		)
	}
	return result
}

// FormatLapsWithElevation returns a formatted string of laps with elevation data
func FormatLapsWithElevation(laps []Lap, streams *Streams) string {
	result := "Lap | Time | Dist | Pace | Avg HR | Max HR | Gain | Loss | Net\n"
	result += "---|------|------|------|--------|-------|------|------|-----\n"

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

		var gain, loss, net int
		if streams != nil && len(streams.Altitude.Data) > 0 {
			lapElev := LapElevationFromStreams(lp, streams.Altitude.Data)
			gain = lapElev.Gain
			loss = lapElev.Loss
			net = lapElev.Net
		}

		result += fmt.Sprintf("%d | %s | %.2f | %s | %s | %s | %+d | %+d | %+d\n",
			lp.LapIndex,
			SecToHHMM(lp.MovingTime),
			distKm,
			PaceFromMoving(lp.Distance, lp.MovingTime),
			avgHR, maxHR,
			gain, loss, net,
		)
	}
	return result
}
