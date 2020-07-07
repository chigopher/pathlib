package pathlib

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

// Path is an object that represents a path
type Path struct {
	path string
	fs   afero.Fs

	// DefaultFileMode is the mode that is used when creating new files in functions
	// that do not accept os.FileMode as a parameter.
	DefaultFileMode os.FileMode
}

// NewPath returns a new OS path
func NewPath(path string) *Path {
	return NewPathAfero(path, afero.NewOsFs())
}

// NewPathAfero returns a Path object with the given Afero object
func NewPathAfero(path string, fs afero.Fs) *Path {
	return &Path{
		path:            path,
		fs:              fs,
		DefaultFileMode: 0o644,
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

// Fs returns the internal afero.Fs object.
func (p *Path) Fs() afero.Fs {
	return p.fs
}

func (p *Path) doesNotImplementErr(interfaceName string) error {
	return errors.Wrapf(ErrDoesNotImplement, "Path's afero filesystem %s does not implement %s", getFsName(p.fs), interfaceName)
}

// *******************************
// * afero.Fs wrappers           *
// *******************************

// Create creates a file if possible, returning the file and an error, if any happens.
func (p *Path) Create() (afero.File, error) {
	return p.Fs().Create(p.Path())
}

// Mkdir makes the current dir. If the parents don't exist, an error
// is returned.
func (p *Path) Mkdir(perm os.FileMode) error {
	return p.Fs().Mkdir(p.Path(), perm)
}

// MkdirAll makes all of the directories up to, and including, the given path.
func (p *Path) MkdirAll(perm os.FileMode) error {
	return p.Fs().MkdirAll(p.Path(), perm)
}

// Open opens a file for read-only, returning it or an error, if any happens.
func (p *Path) Open() (*File, error) {
	handle, err := p.Fs().Open(p.Path())
	return &File{
		File: handle,
	}, err
}

// OpenFile opens a file using the given flags and the given mode.
// See the list of flags at: https://golang.org/pkg/os/#pkg-constants
func (p *Path) OpenFile(flag int, perm os.FileMode) (*File, error) {
	handle, err := p.Fs().OpenFile(p.Path(), flag, perm)
	return &File{
		File: handle,
	}, err
}

// Remove removes a file, returning an error, if any
// happens.
func (p *Path) Remove() error {
	return p.Fs().Remove(p.Path())
}

// RemoveAll removes the given path and all of its children.
func (p *Path) RemoveAll() error {
	return p.Fs().RemoveAll(p.Path())
}

// Rename renames a file
func (p *Path) Rename(newname string) error {
	if err := p.Fs().Rename(p.Path(), newname); err != nil {
		return err
	}

	// Rename succeeded. Set our path to the newname.
	p.path = newname
	return nil
}

// RenamePath is the same as Rename except the argument is a Path object. The attributes
// of the path object is retained and does not inherit anything from target.
func (p *Path) RenamePath(target *Path) error {
	return p.Rename(target.Path())
}

// Stat returns the os.FileInfo of the given path
func (p *Path) Stat() (os.FileInfo, error) {
	return p.Fs().Stat(p.Path())
}

// Chmod changes the file mode of the given path
func (p *Path) Chmod(mode os.FileMode) error {
	return p.Fs().Chmod(p.Path(), mode)
}

// Chtimes changes the modification and access time of the given path.
func (p *Path) Chtimes(atime time.Time, mtime time.Time) error {
	return p.Fs().Chtimes(p.Path(), atime, mtime)
}

// ************************
// * afero.Afero wrappers *
// ************************

// DirExists returns whether or not the path represents a directory that exists
func (p *Path) DirExists() (bool, error) {
	return afero.DirExists(p.Fs(), p.Path())
}

// Exists returns whether the path exists
func (p *Path) Exists() (bool, error) {
	return afero.Exists(p.Fs(), p.Path())
}

// FileContainsAnyBytes returns whether or not the path contains
// any of the listed bytes.
func (p *Path) FileContainsAnyBytes(subslices [][]byte) (bool, error) {
	return afero.FileContainsAnyBytes(p.Fs(), p.Path(), subslices)
}

// FileContainsBytes returns whether or not the given file contains the bytes
func (p *Path) FileContainsBytes(subslice []byte) (bool, error) {
	return afero.FileContainsBytes(p.Fs(), p.Path(), subslice)
}

// IsDir checks if a given path is a directory.
func (p *Path) IsDir() (bool, error) {
	return afero.IsDir(p.Fs(), p.Path())
}

// IsEmpty checks if a given file or directory is empty.
func (p *Path) IsEmpty() (bool, error) {
	return afero.IsEmpty(p.Fs(), p.Path())
}

// ReadDir reads the current path and returns a list of the corresponding
// Path objects.
func (p *Path) ReadDir() ([]*Path, error) {
	var paths []*Path
	fileInfos, err := afero.ReadDir(p.Fs(), p.Path())
	for _, fileInfo := range fileInfos {
		paths = append(paths, p.Join(fileInfo.Name()))
	}
	return paths, err
}

// ReadFile reads the given path and returns the data. If the file doesn't exist
// or is a directory, an error is returned.
func (p *Path) ReadFile() ([]byte, error) {
	return afero.ReadFile(p.Fs(), p.Path())
}

// SafeWriteReader is the same as WriteReader but checks to see if file/directory already exists.
func (p *Path) SafeWriteReader(r io.Reader) error {
	return afero.SafeWriteReader(p.Fs(), p.Path(), r)
}

// Walk walks path, using the given filepath.WalkFunc to handle each
func (p *Path) Walk(walkFn filepath.WalkFunc) error {
	return afero.Walk(p.Fs(), p.Path(), walkFn)
}

// WriteFile writes the given data to the path (if possible). If the file exists,
// the file is truncated. If the file is a directory, or the path doesn't exist,
// an error is returned.
func (p *Path) WriteFile(data []byte, perm os.FileMode) error {
	return afero.WriteFile(p.Fs(), p.Path(), data, perm)
}

// WriteReader takes a reader and writes the content
func (p *Path) WriteReader(r io.Reader) error {
	return afero.WriteReader(p.Fs(), p.Path(), r)
}

// *************************************
// * pathlib.Path-like implementations *
// *************************************

// Name returns the string representing the final path component
func (p *Path) Name() string {
	return filepath.Base(p.path)
}

// Parent returns the Path object of the parent directory
func (p *Path) Parent() *Path {
	return NewPathAfero(filepath.Dir(p.Path()), p.Fs())
}

// Resolve resolves the path to a real path
func (p *Path) Resolve() (*Path, error) {
	linkReader, ok := p.Fs().(afero.LinkReader)
	if !ok {
		return nil, p.doesNotImplementErr("afero.LinkReader")
	}

	resolvedPathStr, err := linkReader.ReadlinkIfPossible(p.path)
	if err != nil {
		return nil, err
	}
	return NewPathAfero(resolvedPathStr, p.fs), nil
}

// IsAbsolute returns whether or not the path is an absolute path. This is
// determined by checking if the path starts with a slash.
func (p *Path) IsAbsolute() bool {
	return strings.HasPrefix(p.path, "/")
}

// Join joins the current object's path with the given elements and returns
// the resulting Path object.
func (p *Path) Join(elems ...string) *Path {
	paths := []string{p.path}
	for _, path := range elems {
		paths = append(paths, path)
	}
	return NewPathAfero(filepath.Join(paths...), p.fs)
}

func normalizePathString(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimRight(path, " ")
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func normalizePathParts(path []string) []string {
	// We might encounter cases where path represents a split of the path
	// "///" etc. We will get a bunch of erroneous empty strings in such a split,
	// so remove all of the trailing empty strings except for the first one (if any)
	for i := len(path) - 1; i > 0; i-- {
		if path[i] == "" {
			path = path[0:i]
		} else {
			break
		}
	}
	return path
}

// RelativeTo computes a relative version of path to the other path. For instance,
// if the object is /path/to/foo.txt and you provide /path/ as the argment, the
// returned Path object will represent to/foo.txt.
func (p *Path) RelativeTo(other *Path) (*Path, error) {
	thisPath := normalizePathString(p.Path())
	otherPath := normalizePathString(other.Path())

	thisParts := normalizePathParts(strings.Split(thisPath, string(filepath.Separator)))
	otherParts := normalizePathParts(strings.Split(otherPath, string(filepath.Separator)))

	relativePath := []string{}
	var relativeBase int
	for idx, part := range otherParts {
		if thisParts[idx] != part {
			return p, errors.Errorf("%s does not start with %s", p.path, otherPath)
		}
		relativeBase = idx
	}

	relativePath = thisParts[relativeBase+1:]

	if len(relativePath) == 0 || (len(relativePath) == 1 && relativePath[0] == "") {
		relativePath = []string{"."}
	}

	return NewPathAfero(strings.Join(relativePath, "/"), p.Fs()), nil
}

// *********************************
// * filesystem-specific functions *
// *********************************

// Symlink symlinks to the target location. This will fail if the underlying
// afero filesystem does not implement afero.Linker.
func (p *Path) Symlink(target *Path) error {
	symlinker, ok := p.fs.(afero.Linker)
	if !ok {
		return p.doesNotImplementErr("afero.Linker")
	}

	return symlinker.SymlinkIfPossible(target.path, p.path)
}

// ****************************************
// * chigopher/pathlib-specific functions *
// ****************************************

// IsFile returns true if the given path is a file.
func (p *Path) IsFile() (bool, error) {
	fileInfo, err := p.Fs().Stat(p.Path())
	if err != nil {
		return false, err
	}
	return fileInfo.Mode().IsRegular(), nil
}

// IsSymlink returns true if the given path is a symlink.
// Fails if the filesystem doesn't implement afero.Lstater.
func (p *Path) IsSymlink() (bool, error) {
	lStater, ok := p.Fs().(afero.Lstater)
	if !ok {
		return false, p.doesNotImplementErr("afero.Lstater")
	}
	fileInfo, lstatCalled, err := lStater.LstatIfPossible(p.Path())
	if err != nil || !lstatCalled {
		// If lstat wasn't called then the filesystem doesn't implement it.
		// Thus, it isn't a symlink
		return false, err
	}

	isSymlink := false
	if fileInfo.Mode()&os.ModeSymlink != 0 {
		isSymlink = true
	}
	return isSymlink, nil
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
	return Glob(p.Fs(), p.Join(pattern).Path())
}

// Mtime returns the modification time of the given path.
func (p *Path) Mtime() (time.Time, error) {
	stat, err := p.Stat()
	if err != nil {
		return time.Time{}, err
	}
	return stat.ModTime(), nil
}
