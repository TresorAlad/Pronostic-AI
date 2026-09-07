package db

import "time"

// matchStatusWhere builds SQL conditions so past matches do not reappear in top lists or coupons.
func matchStatusWhere(status string, queryDate time.Time) string {
	day := queryDate.UTC().Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")

	switch status {
	case "live":
		return `m.status = 'live'`
	case "finished":
		return `m.status = 'finished'`
	case "scheduled":
		if day > today {
			return `m.status = 'scheduled'`
		}
		return `m.status = 'scheduled' AND m.kickoff_at > NOW()`
	default:
		if day < today {
			return `m.status = 'finished'`
		}
		if day > today {
			return `m.status = 'scheduled'`
		}
		return `(m.status = 'live' OR (m.status = 'scheduled' AND m.kickoff_at > NOW()))`
	}
}

// couponMatchWhere keeps only upcoming scheduled matches (future kickoff).
func couponMatchWhere() string {
	return `m.status = 'scheduled' AND m.kickoff_at > NOW()`
}

// upcomingMatchCountWhere counts playable matches for league filters on a given day.
func upcomingMatchCountWhere(queryDate time.Time) string {
	day := queryDate.UTC().Format("2006-01-02")
	today := time.Now().UTC().Format("2006-01-02")
	if day < today {
		return `m.status = 'finished'`
	}
	if day > today {
		return `m.status = 'scheduled'`
	}
	return `(m.status = 'live' OR (m.status = 'scheduled' AND m.kickoff_at > NOW()))`
}
