package storaged

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

func Quota(filePath string) (int, error) {
	var output [128]byte
	sz, err := unix.Getxattr(filePath, "ceph.quota.max_bytes", output[:])
	if err != nil {
		return 0, fmt.Errorf("error getting xattr: %w", err)
	}
	if sz == 0 {
		// This directory is not limited.
		// A reasonable number that wouldn't overflow but would basically max
		// out the limit.
		return 1 << 50, nil
	}
	rawQuotaStr := string(output[:sz])
	quotaStr := rawQuotaStr
	multiplier := 1
	switch {
	case strings.HasSuffix(quotaStr, "K"):
		quotaStr = strings.TrimSuffix(quotaStr, "K")
		multiplier = 1024
	case strings.HasSuffix(quotaStr, "Ki"):
		quotaStr = strings.TrimSuffix(quotaStr, "Ki")
		multiplier = 1024
	case strings.HasSuffix(quotaStr, "M"):
		quotaStr = strings.TrimSuffix(quotaStr, "M")
		multiplier = 1024 * 1024
	case strings.HasSuffix(quotaStr, "Mi"):
		quotaStr = strings.TrimSuffix(quotaStr, "Mi")
		multiplier = 1024 * 1024
	case strings.HasSuffix(quotaStr, "G"):
		quotaStr = strings.TrimSuffix(quotaStr, "G")
		multiplier = 1024 * 1024 * 1024
	case strings.HasSuffix(quotaStr, "Gi"):
		quotaStr = strings.TrimSuffix(quotaStr, "Gi")
		multiplier = 1024 * 1024 * 1024
	}
	quotaNum, err := strconv.Atoi(strings.TrimSpace(quotaStr))
	if err != nil {
		return 0, fmt.Errorf("unknown quota size: %s", rawQuotaStr)
	}
	return quotaNum * multiplier, nil
}

func SetQuota(filePath string, maxBytes int) error {
	return unix.Setxattr(filePath, "ceph.quota.max_bytes", []byte(strconv.Itoa(maxBytes)), 0)
}
