package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ftpserver "github.com/goftp/server"
)

var errReadOnly = errors.New("file server is read-only")

type fileDriver struct {
	root     string
	readOnly bool
	owner    string
}

func newFileDriver(root string, readOnly bool) *fileDriver {
	return &fileDriver{root: filepath.Clean(root), readOnly: readOnly, owner: "ftp"}
}

func (d *fileDriver) resolvePath(virtualPath string) (string, error) {
	virtualPath = strings.ReplaceAll(virtualPath, "/", string(filepath.Separator))
	candidate := filepath.Join(d.root, virtualPath)
	relative, err := filepath.Rel(d.root, candidate)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes FTP root")
	}
	rootReal, err := canonicalPath(d.root)
	if err != nil {
		return "", fmt.Errorf("resolve FTP root: %w", err)
	}
	existing := candidate
	for {
		if _, err := os.Lstat(existing); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", errors.New("unable to resolve path")
		}
		existing = parent
	}
	existingReal, err := canonicalPath(existing)
	if err != nil {
		return "", err
	}
	relative, err = filepath.Rel(rootReal, existingReal)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("path resolves outside FTP root")
	}
	return candidate, nil
}

func canonicalPath(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	// Windows may deny reparse-point inspection for otherwise usable folders.
	// Lexical confinement remains in force in that environment.
	if errors.Is(err, os.ErrPermission) {
		return filepath.Clean(path), nil
	}
	return "", err
}

func (d *fileDriver) requireWritable() error {
	if d.readOnly {
		return errReadOnly
	}
	return nil
}

func (d *fileDriver) Init(*ftpserver.Conn) {}

func (d *fileDriver) Stat(path string) (ftpserver.FileInfo, error) {
	realPath, err := d.resolvePath(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return nil, err
	}
	return fileInfo{FileInfo: info, owner: d.owner}, nil
}

func (d *fileDriver) ChangeDir(path string) error {
	realPath, err := d.resolvePath(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(realPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("not a directory")
	}
	return nil
}

func (d *fileDriver) ListDir(path string, callback func(ftpserver.FileInfo) error) error {
	realPath, err := d.resolvePath(path)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(realPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := callback(fileInfo{FileInfo: info, owner: d.owner}); err != nil {
			return err
		}
	}
	return nil
}

func (d *fileDriver) DeleteDir(path string) error {
	if err := d.requireWritable(); err != nil {
		return err
	}
	realPath, err := d.resolvePath(path)
	if err != nil {
		return err
	}
	return os.Remove(realPath)
}

func (d *fileDriver) DeleteFile(path string) error {
	if err := d.requireWritable(); err != nil {
		return err
	}
	realPath, err := d.resolvePath(path)
	if err != nil {
		return err
	}
	return os.Remove(realPath)
}

func (d *fileDriver) Rename(fromPath, toPath string) error {
	if err := d.requireWritable(); err != nil {
		return err
	}
	from, err := d.resolvePath(fromPath)
	if err != nil {
		return err
	}
	to, err := d.resolvePath(toPath)
	if err != nil {
		return err
	}
	return os.Rename(from, to)
}

func (d *fileDriver) MakeDir(path string) error {
	if err := d.requireWritable(); err != nil {
		return err
	}
	realPath, err := d.resolvePath(path)
	if err != nil {
		return err
	}
	return os.Mkdir(realPath, 0755)
}

func (d *fileDriver) GetFile(path string, offset int64) (int64, io.ReadCloser, error) {
	realPath, err := d.resolvePath(path)
	if err != nil {
		return 0, nil, err
	}
	file, err := os.Open(realPath)
	if err != nil {
		return 0, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return 0, nil, err
	}
	if offset < 0 || offset > info.Size() {
		file.Close()
		return 0, nil, errors.New("file offset is outside the file")
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		file.Close()
		return 0, nil, err
	}
	return info.Size() - offset, file, nil
}

func (d *fileDriver) PutFile(path string, data io.Reader, appendData bool) (int64, error) {
	if err := d.requireWritable(); err != nil {
		return 0, err
	}
	realPath, err := d.resolvePath(path)
	if err != nil {
		return 0, err
	}
	flags := os.O_WRONLY | os.O_CREATE
	if appendData {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(realPath, flags, 0644)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return io.Copy(file, data)
}

type fileInfo struct {
	os.FileInfo
	owner string
}

func (info fileInfo) Owner() string { return info.owner }
func (info fileInfo) Group() string { return info.owner }

type fileDriverFactory struct {
	root     string
	readOnly bool
	owner    string
}

func (factory fileDriverFactory) NewDriver() (ftpserver.Driver, error) {
	return &fileDriver{root: factory.root, readOnly: factory.readOnly, owner: factory.owner}, nil
}
