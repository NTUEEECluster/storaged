package storaged

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// CephFS is an implementation of QuotaFS based on actual Ceph filesystem.
type CephFS struct {
	rootFS
	LinkUID int
	LinkGID int
}

type rootFS fs.FS

var _ QuotaFS = (*CephFS)(nil)

func NewCephFS(linkUID int, linkGID int) (*CephFS, error) {
	rootFS, err := os.OpenRoot("/")
	if err != nil {
		return nil, fmt.Errorf("error opening root: %w", err)
	}
	return &CephFS{
		rootFS:  rootFS.FS(),
		LinkUID: linkUID,
		LinkGID: linkGID,
	}, nil
}

func (fs CephFS) FileOwner(filePath string) (string, error) {
	var output unix.Stat_t
	err := unix.Stat(filePath, &output)
	if err != nil {
		return "", fmt.Errorf("error getting file stat: %w", err)
	}
	userInfo, err := user.LookupId(strconv.Itoa(int(output.Uid)))
	if err != nil {
		return "", fmt.Errorf("error getting info for owner of %s: %w", filePath, err)
	}
	return userInfo.Username, nil
}

func (fs CephFS) Usage(filePath string) (int, error) {
	var output [128]byte
	sz, err := unix.Getxattr(filePath, "ceph.dir.rbytes", output[:])
	if err != nil {
		return 0, fmt.Errorf("error getting xattr: %w", err)
	}
	usageStr := string(output[:sz])
	return strconv.Atoi(usageStr)
}

func (fs CephFS) Quota(filePath string) (int, error) {
	var output [128]byte
	sz, err := unix.Getxattr(filePath, "ceph.quota.max_bytes", output[:])
	if err != nil {
		return 0, fmt.Errorf("error getting xattr: %w", err)
	}
	if sz == 0 {
		return QuotaUnbounded, nil
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

func (fs CephFS) SetQuota(filePath string, maxBytes int) error {
	return unix.Setxattr(filePath, "ceph.quota.max_bytes", []byte(strconv.Itoa(maxBytes)), 0)
}

func (fs CephFS) CreateLink(filePath string, absoluteTarget string) error {
	err := os.Symlink(filePath, absoluteTarget)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	err = os.Chown(filePath, fs.LinkUID, fs.LinkGID)
	if err != nil {
		defer os.Remove(filePath)
		return fmt.Errorf("failed to chown symlink: %w", err)
	}
	return nil
}

func (cephFS CephFS) CreateFolder(filePath string, uid, gid string) error {
	uidNum, err := strconv.Atoi(uid)
	if err != nil {
		return fmt.Errorf("error parsing UID %q: %w", uid, err)
	}
	gidNum, err := strconv.Atoi(gid)
	if err != nil {
		return fmt.Errorf("error parsing GID %q: %w", uid, err)
	}
	err = os.Mkdir(filePath, 0o770)
	if err != nil {
		return fmt.Errorf("error creating folder: %w", err)
	}
	err = os.Chown(filePath, uidNum, gidNum)
	if err != nil {
		os.Remove(filePath)
		return fmt.Errorf("error chown-ing folder: %w", err)
	}
	return nil
}

func (cephFS CephFS) DeleteFolder(filePath string) error {
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("error removing directory: %w", err)
	}
	return nil
}

func (cephFS CephFS) PathFor(filePath string) string {
	return path.Clean(filePath)
}
