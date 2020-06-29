package pathlib

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// Path is an object that represents a path
type Path struct {
	path  string
	afero afero.Afero
}

// NewPath returns a new OS path
func NewPath(path string) *Path {
	return NewPathAfero(path, afero.NewOsFs())
}

// NewPathAfero returns a Path object with the given Afero object
func NewPathAfero(path string, fs afero.Fs) *Path {
	return &Path{
		path:  path,
		afero: afero.Afero{Fs: fs},
	}
}

// Glob returns all of the path objects matched by the given pattern
// inside of the afero filesystem.
func Glob(fs afero.Fs, pattern string) ([]*Path, error) {
	matches, err := afero.Glob(fs, pattern)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to glob")
	}

	pathMatches := []*Path{}
	for _, match := range matches {
		pathMatches = append(pathMatches, NewPathAfero(match, fs))
	}
	return pathMatches, nil
}

type namer interface {
	Name() string
}

func getFsName(fs afero.Fs) string {
	if name, ok := fs.(namer); ok {
		return name.Name()
	}
	return ""
}

// Afero returns the internal afero object.
func (p *Path) Afero() afero.Afero {
	return p.afero
}

func (p *Path) doesNotImplementErr(interfaceName string) error {
	return errors.Wrapf(ErrDoesNotImplement, "Path's afero filesystem %s does not implement %s", getFsName(p.afero.Fs), interfaceName)
}

// Resolve resolves the path to a real path
func (p *Path) Resolve() (*Path, error) {
	linkReader, ok := p.afero.Fs.(afero.LinkReader)
	if !ok {
		return nil, p.doesNotImplementErr("afero.LinkReader")
	}

	resolvedPathStr, err := linkReader.ReadlinkIfPossible(p.path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to readlink")
	}
	return NewPathAfero(resolvedPathStr, p.afero.Fs), nil
}

// Symlink symlinks to the target location
func (p *Path) Symlink(target *Path) error {
	symlinker, ok := p.afero.Fs.(afero.Linker)
	if !ok {
		return p.doesNotImplementErr("afero.Linker")
	}

	return errors.Wrapf(symlinker.SymlinkIfPossible(target.path, p.path), "failed to symlink %s to %s", p.path, target.path)
}

// IsAbsolute returns whether or not the path is an absolute path. This is
// determined by checking if the path starts with a slash.
func (p *Path) IsAbsolute() bool {
	return strings.HasPrefix(p.path, "/")
}

// Name returns the string representing the final path component
func (p *Path) Name() string {
	return filepath.Base(p.path)
}

// Join joins the current object's path with the given elements and returns
// the resulting Path object.
func (p *Path) Join(elems ...string) *Path {
	paths := []string{p.path}
	for _, path := range elems {
		paths = append(paths, path)
	}
	return NewPathAfero(filepath.Join(paths...), p.afero.Fs)
}

// WriteFile writes the given data to the path (if possible). If the file exists,
// the file is truncated. If the file is a directory, or the path doesn't exist,
// an error is returned.
func (p *Path) WriteFile(data []byte, perm os.FileMode) error {
	return errors.Wrapf(p.afero.WriteFile(p.Path(), data, perm), "Failed to write file")
}

// ReadFile reads the given path and returns the data. If the file doesn't exist
// or is a directory, an error is returned.
func (p *Path) ReadFile() ([]byte, error) {
	bytes, err := p.afero.ReadFile(p.Path())
	return bytes, errors.Wrapf(err, "failed to read file")
}

// ReadDir reads the current path and returns a list of the corresponding
// Path objects.
func (p *Path) ReadDir() ([]*Path, error) {
	var paths []*Path
	fileInfos, err := p.afero.ReadDir(p.Path())
	for _, fileInfo := range fileInfos {
		paths = append(paths, p.Join(fileInfo.Name()))
	}
	return paths, errors.Wrapf(err, "failed to read directory")
}

// chmoder should really be part of afero. TODO: Send a PR to upstream
type chmoder interface {
	Chmod(name string, mode os.FileMode) error
}

// Chmod changes the file mode of the given path
func (p *Path) Chmod(mode os.FileMode) error {
	chmodCaller, ok := p.afero.Fs.(chmoder)
	if !ok {
		return p.doesNotImplementErr("Chmod")
	}

	return errors.Wrapf(chmodCaller.Chmod(p.path, mode), "Failed to chmod")
}

type mkdir interface {
	Mkdir(name string, perm os.FileMode) error
}

// Mkdir makes the current dir. If the parents don't exist, an error
// is returned.
func (p *Path) Mkdir(perm os.FileMode) error {
	mkdirCaller, ok := p.afero.Fs.(mkdir)
	if !ok {
		return p.doesNotImplementErr("Mkdir")
	}
	return errors.Wrapf(mkdirCaller.Mkdir(p.path, perm), "failed to Mkdir")
}

type mkdirAll interface {
	MkdirAll(name string, perm os.FileMode) error
}

// MkdirAll makes all of the directories up to, and including, the given path.
func (p *Path) MkdirAll(perm os.FileMode) error {
	mkdirCaller, ok := p.afero.Fs.(mkdirAll)
	if !ok {
		return p.doesNotImplementErr("MkdirAll")
	}
	return errors.Wrapf(mkdirCaller.MkdirAll(p.path, perm), "failed to Mkdir")
}

type rename interface {
	Rename(oldname, newname string) error
}

// Rename this path to the given target and return the corresponding
// Path object.
func (p *Path) Rename(target string) (*Path, error) {
	renameCaller, ok := p.afero.Fs.(rename)
	if !ok {
		return nil, p.doesNotImplementErr("Rename")
	}

	err := errors.Wrapf(renameCaller.Rename(p.path, target), "failed to rename")
	if err != nil {
		return nil, err
	}
	return NewPathAfero(target, p.afero.Fs), nil
}

// RenamePath is the same as Rename except target is a Path object
func (p *Path) RenamePath(target *Path) (*Path, error) {
	return p.Rename(target.path)
}

type remover interface {
	Remove(name string) error
}

// Remove deletes/unlinks/destroys the given path
func (p *Path) Remove() error {
	removeCaller, ok := p.afero.Fs.(remover)
	if !ok {
		return p.doesNotImplementErr("Remove")
	}

	return errors.Wrapf(removeCaller.Remove(p.path), "failed to remove")
}

type removeAll interface {
	RemoveAll(name string) error
}

// RemoveAll removes the given path and all of its children.
func (p *Path) RemoveAll() error {
	removeAllCaller, ok := p.afero.Fs.(removeAll)
	if !ok {
		return p.doesNotImplementErr("RemoveAll")
	}

	return errors.Wrapf(removeAllCaller.RemoveAll(p.path), "failed to remove all")
}

// Exists returns whether the path exists
func (p *Path) Exists() (bool, error) {
	return p.afero.Exists(p.path)
}

// IsDir returns whether the path is a directory
func (p *Path) IsDir() (bool, error) {
	return p.afero.IsDir(p.path)
}

// IsFile returns true if the given path is a file.
func (p *Path) IsFile() (bool, error) {
	fileInfo, err := p.afero.Stat(p.path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to stat")
	}
	return fileInfo.Mode().IsRegular(), nil
}

// IsSymlink returns true if the given path is a symlink.
func (p *Path) IsSymlink() (bool, error) {
	fileInfo, err := p.afero.Stat(p.path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to stat")
	}

	isSymlink := false
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		isSymlink = true
	}
	return isSymlink, nil
}

// Stat returns the os.FileInfo of the given path
func (p *Path) Stat() (os.FileInfo, error) {
	return p.afero.Stat(p.path)
}

// Parent returns the Path object of the parent directory
func (p *Path) Parent() *Path {
	return NewPathAfero(filepath.Dir(p.path), p.afero.Fs)
}

// Path returns the string representation of the path
func (p *Path) Path() string {
	return p.path
}

// Equals returns whether or not the path pointed to by other
// has the same resolved filepath as self.
func (p *Path) Equals(other *Path) (bool, error) {
	selfResolved, err := p.Resolve()
	if err != nil {
		return false, err
	}
	otherResolved, err := other.Resolve()
	if err != nil {
		return false, err
	}

	return selfResolved.Path() == otherResolved.Path(), nil
}

// RelativeTo computes a relative version of path to the other path. For instance,
// if the object is /path/to/foo.txt and you provide /path/ as the argment, the
// returned Path object will represent to/foo.txt.
func (p *Path) RelativeTo(other *Path) (*Path, error) {
	thisParts := strings.Split(p.path, "/")
	// Normalize
	if thisParts[len(thisParts)-1] == "" {
		thisParts = thisParts[:len(thisParts)-1]
	}
	if thisParts[0] == "." {
		thisParts = thisParts[1:]
	}

	otherParts := strings.Split(other.path, "/")
	// Normalize
	if len(otherParts) > 1 && otherParts[len(otherParts)-1] == "" {
		otherParts = otherParts[:len(otherParts)-1]
	}
	if otherParts[0] == "." {
		otherParts = otherParts[1:]
	}

	if !strings.HasPrefix(p.path, other.path) {
		errors.Errorf("%s does not start with %s", p.path, other.path)
	}

	relativePath := []string{}
	var relativeBase int
	for idx, part := range otherParts {
		if thisParts[idx] != part {
			return nil, errors.Errorf("%s does not start with %s", p.path, strings.Join(otherParts[:idx], "/"))
		}
		relativeBase = idx
	}

	relativePath = thisParts[relativeBase+1:]

	if len(relativePath) == 0 || (len(relativePath) == 1 && relativePath[0] == "") {
		relativePath = []string{"."}
	}

	return NewPathAfero(strings.Join(relativePath, "/"), p.afero.Fs), nil
}

// GetLatest returns the file or directory that has the most recent mtime. Only
// works if this path is a directory and it exists. If the directory is empty,
// the returned Path object will be nil.
func (p *Path) GetLatest() (*Path, error) {
	files, err := p.ReadDir()
	if err != nil {
		return nil, err
	}

	var greatestFileSeen *Path
	for _, file := range files {
		if greatestFileSeen == nil {
			greatestFileSeen = p.Join(file.Name())
		}

		greatestMtime, err := greatestFileSeen.Mtime()
		if err != nil {
			return nil, err
		}

		thisMtime, err := file.Mtime()
		// There is a possible race condition where the file is deleted after
		// our call to ReadDir. We throw away the error if it isn't
		// os.ErrNotExist
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if thisMtime.After(greatestMtime) {
			greatestFileSeen = p.Join(file.Name())
		}
	}

	return greatestFileSeen, nil
}

// Glob returns all matches of pattern relative to this object's path.
func (p *Path) Glob(pattern string) ([]*Path, error) {
	return Glob(p.afero.Fs, p.Join(pattern).Path())
}

// Chtimes changes the modification and access time of the given path.
func (p *Path) Chtimes(atime time.Time, mtime time.Time) error {
	return p.afero.Chtimes(p.Path(), atime, mtime)
}

// Mtime returns the modification time of the given path.
func (p *Path) Mtime() (time.Time, error) {
	stat, err := p.Stat()
	if err != nil {
		return time.Time{}, err
	}
	return stat.ModTime(), nil
}
