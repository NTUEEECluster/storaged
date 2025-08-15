package storaged

import (
	"fmt"
	"io/fs"
)

// QuotaUnbounded is returned if the directory is unbounded in quota. It is a
// reasonable number that wouldn't overflow but would basically max out the
// limit.
var QuotaUnbounded = 1 << 50

type Quota struct {
	Name  string
	Usage int
	Quota int
}

// QuotaUsed returns the quota allocation used by the user.
func QuotaUsed(quotaFS QuotaFS, user string) ([]Quota, int, error) {
	entries, err := fs.ReadDir(quotaFS, ".")
	if err != nil {
		return nil, 0, fmt.Errorf("error reading directory entries: %w", err)
	}
	quotaEntries := []Quota{}
	quotaUsed := 0
	for _, entry := range entries {
		name := entry.Name()
		owner, err := quotaFS.FileOwner(name)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get owner of %q: %w", name, err)
		}
		if owner != user {
			continue
		}
		usage, err := quotaFS.Usage(name)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get usage assigned to %q: %w", name, err)
		}
		quota, err := quotaFS.Quota(name)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get quota assigned to %q: %w", name, err)
		}
		quotaUsed += quota
		quotaEntries = append(quotaEntries, Quota{
			Name:  name,
			Usage: usage,
			Quota: quota,
		})
	}
	return quotaEntries, quotaUsed, nil
}
