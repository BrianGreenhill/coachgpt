package strava

import (
	"fmt"
	"math"
)

// SplitHR represents heart rate data for a split
type SplitHR struct {
	KM        int
	Pace      string
	AvgHR     int
	MaxHR     int
	ElevDelta int
}

// LapElev represents elevation data for a lap
type LapElev struct {
	Gain int
	Loss int
	Net  int
}

// SecToHHMM converts seconds to HH:MM format
func SecToHHMM(sec int64) string {
	m := sec / 60
	h := m / 60
	m = m % 60
	return fmt.Sprintf("%d:%02d", h, m)
}

// PaceFromMoving calculates pace from distance and moving time
func PaceFromMoving(distanceMeters float64, movingSec int64) string {
	if distanceMeters <= 0 || movingSec <= 0 {
		return "-"
	}
	secPerKm := float64(movingSec) / (distanceMeters / 1000.0) // sec/km
	m := int(secPerKm) / 60
	s := int(secPerKm) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

// ComputeZones calculates heart rate zone distribution
func ComputeZones(hr []float64, hrmax int) (z [5]int) {
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

// ComputeSplitHR calculates heart rate data for each split
func ComputeSplitHR(splits []Split, timeStream, hrStream []float64) []SplitHR {
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
			Pace:      PaceFromMoving(sp.Distance, sp.MovingTime),
			AvgHR:     avg,
			MaxHR:     max,
			ElevDelta: int(math.Round(sp.ElevationDifference)),
		})
	}
	return out
}

// LapElevationFromStreams calculates elevation data from altitude stream
func LapElevationFromStreams(l Lap, altitude []float64) LapElev {
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
