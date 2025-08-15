package storaged

import (
	"fmt"
	"io/fs"
	"path"
)

type QuotaFS interface {
	fs.FS

	// Usage returns the current usage for the file path.
	Usage(project string) (int, error)
	// Quota returns the current Quota set for the file path.
	Quota(project string) (int, error)
	// SetQuota sets the Quota for the file path.
	SetQuota(project string, newQuota int) error
	// FileOwner returns the name of the owner of the file.
	FileOwner(project string) (string, error)

	// CreateFolder creates the specified folder.
	CreateFolder(project string, uid string, gid string) error
	// DeleteFolder deletes the specified folder.
	DeleteFolder(project string) error
	// NewLink creates a link from the project name to the specified location.
	CreateLink(project string, absoluteTarget string) error

	// PathFor returns the absolute path for the specified project.
	PathFor(project string) string
}

func SubFS(quotaFS QuotaFS, subPath string) (QuotaFS, error) {
	subRawFS, err := fs.Sub(quotaFS, subPath)
	if err != nil {
		return nil, fmt.Errorf("cannot get sub-FS: %w", err)
	}
	return &subQuotaFS{
		FS:       subRawFS,
		original: quotaFS,
		path:     subPath,
	}, nil
}

type subQuotaFS struct {
	fs.FS
	original QuotaFS
	path     string
}

func (f *subQuotaFS) Usage(filepath string) (int, error) {
	if !fs.ValidPath(filepath) {
		return 0, fmt.Errorf("cannot get usage of invalid path %s", filepath)
	}
	return f.original.Usage(path.Join(f.path, filepath))
}

func (f *subQuotaFS) Quota(filepath string) (int, error) {
	if !fs.ValidPath(filepath) {
		return 0, fmt.Errorf("cannot get quota of invalid path %s", filepath)
	}
	return f.original.Quota(path.Join(f.path, filepath))
}

func (f *subQuotaFS) SetQuota(filepath string, newQuota int) error {
	if !fs.ValidPath(filepath) {
		return fmt.Errorf("cannot set quota of invalid path %s", filepath)
	}
	return f.original.SetQuota(path.Join(f.path, filepath), newQuota)
}

func (f *subQuotaFS) FileOwner(filepath string) (string, error) {
	if !fs.ValidPath(filepath) {
		return "", fmt.Errorf("cannot get owner of invalid path %s", filepath)
	}
	return f.original.FileOwner(path.Join(f.path, filepath))
}

func (f *subQuotaFS) CreateFolder(filepath, uid, gid string) error {
	if !fs.ValidPath(filepath) {
		return fmt.Errorf("cannot create folder of invalid path %s", filepath)
	}
	return f.original.CreateFolder(path.Join(f.path, filepath), uid, gid)
}

func (f *subQuotaFS) DeleteFolder(filepath string) error {
	if !fs.ValidPath(filepath) {
		return fmt.Errorf("cannot delete folder of invalid path %s", filepath)
	}
	return f.original.DeleteFolder(path.Join(f.path, filepath))
}

func (f *subQuotaFS) CreateLink(filepath, absoluteTarget string) error {
	if !fs.ValidPath(filepath) {
		return fmt.Errorf("cannot create link of invalid path %s", filepath)
	}
	return f.original.CreateLink(path.Join(f.path, filepath), absoluteTarget)
}

func (f *subQuotaFS) PathFor(project string) string {
	return f.original.PathFor(path.Join(f.path, project))
}
