package storaged

import (
	"fmt"
	"os/user"
)

func (s *Server) allowedQuota(checkTarget *user.User) (map[string]int, error) {
	gids, err := checkTarget.GroupIds()
	if err != nil {
		return nil, fmt.Errorf("error finding group IDs for user %q: %w", checkTarget.Username, err)
	}
	bestQuota := make(map[string]int)
	for _, gid := range gids {
		group, err := user.LookupGroupId(gid)
		if err != nil {
			return nil, fmt.Errorf("error finding name of group ID %q: %w", gid, err)
		}
		for _, v := range s.Allocations[group.Name] {
			bestQuota[v.Tier] = max(bestQuota[v.Tier], v.MaxBytes)
		}
	}
	return bestQuota, nil
}

func FormatByteSize(byteCount int) string {
	const kilo = 1000
	const mega = kilo * 1000
	const giga = mega * 1000
	const tera = giga * 1000
	switch {
	case byteCount >= QuotaUnbounded:
		return "UNBOUNDED"
	case byteCount < 0:
		return "Negative? (Report Bug!)"
	case byteCount >= tera:
		return fmt.Sprintf("%.1f T", float64(byteCount)/tera)
	case byteCount >= giga:
		return fmt.Sprintf("%.1f G", float64(byteCount)/giga)
	case byteCount >= mega:
		return fmt.Sprintf("%.1f M", float64(byteCount)/mega)
	case byteCount >= kilo:
		return fmt.Sprintf("%.1f K", float64(byteCount)/kilo)
	default:
		return fmt.Sprintf("%d B", byteCount)
	}
}
