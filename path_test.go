package pathlib

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PathSuite struct {
	suite.Suite
	tmpdir *Path
}

func (p *PathSuite) SetupTest() {
	// We actually can't use the MemMapFs because some of the tests
	// are testing symlink behavior. We might want to split these
	// tests out to use MemMapFs when possible.
	tmpdir, err := ioutil.TempDir("", "")
	require.NoError(p.T(), err)
	p.tmpdir = NewPath(tmpdir)
}

func (p *PathSuite) TeardownTest() {
	assert.NoError(p.T(), p.tmpdir.RemoveAll())
}

func (p *PathSuite) TestSymlink() {
	symlink := p.tmpdir.Join("symlink")
	require.NoError(p.T(), symlink.Symlink(p.tmpdir))

	linkLocation, err := symlink.Readlink()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), p.tmpdir.path, linkLocation.path)
}

func (p *PathSuite) TestSymlinkBadFs() {
	symlink := p.tmpdir.Join("symlink")
	symlink.fs = afero.NewMemMapFs()

	assert.Error(p.T(), symlink.Symlink(p.tmpdir))
}

func (p *PathSuite) TestJoin() {
	joined := p.tmpdir.Join("test1")
	assert.Equal(p.T(), filepath.Join(p.tmpdir.String(), "test1"), joined.String())
}

func (p *PathSuite) TestWriteAndRead() {
	expectedBytes := []byte("hello world!")
	file := p.tmpdir.Join("test.txt")
	require.NoError(p.T(), file.WriteFile(expectedBytes))
	bytes, err := file.ReadFile()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), expectedBytes, bytes)
}

func (p *PathSuite) TestChmod() {
	file := p.tmpdir.Join("file1.txt")
	require.NoError(p.T(), file.WriteFile([]byte("")))

	require.NoError(p.T(), file.Chmod(0o777))
	fileInfo, err := file.Stat()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), os.FileMode(0o777), fileInfo.Mode()&os.ModePerm)

	require.NoError(p.T(), file.Chmod(0o755))
	fileInfo, err = file.Stat()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), os.FileMode(0o755), fileInfo.Mode()&os.ModePerm)
}

func (p *PathSuite) TestMkdir() {
	subdir := p.tmpdir.Join("subdir")
	assert.NoError(p.T(), subdir.Mkdir())
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestMkdirParentsDontExist() {
	subdir := p.tmpdir.Join("subdir1", "subdir2")
	assert.Error(p.T(), subdir.Mkdir())
}

func (p *PathSuite) TestMkdirAll() {
	subdir := p.tmpdir.Join("subdir")
	assert.NoError(p.T(), subdir.MkdirAll())
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestMkdirAllMultipleSubdirs() {
	subdir := p.tmpdir.Join("subdir1", "subdir2", "subdir3")
	assert.NoError(p.T(), subdir.MkdirAll())
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestRenameString() {
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte("hello world!")))

	newPath := p.tmpdir.Join("file2.txt")

	err := file.Rename(newPath)
	assert.NoError(p.T(), err)
	assert.Equal(p.T(), file.String(), p.tmpdir.Join("file2.txt").String())

	newBytes, err := file.ReadFile()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), []byte("hello world!"), newBytes)

	newFileExists, err := file.Exists()
	require.NoError(p.T(), err)
	assert.True(p.T(), newFileExists)

	oldFileExists, err := p.tmpdir.Join("file.txt").Exists()
	require.NoError(p.T(), err)
	assert.False(p.T(), oldFileExists)
}

func (p *PathSuite) TestSizeZero() {
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte{}))
	size, err := file.Size()
	require.NoError(p.T(), err)
	p.Zero(size)
}

func (p *PathSuite) TestSizeNonZero() {
	msg := "oh, it's you"
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte(msg)))
	size, err := file.Size()
	require.NoError(p.T(), err)
	p.Equal(len(msg), int(size))
}

func (p *PathSuite) TestIsDir() {
	dir := p.tmpdir.Join("dir")
	require.NoError(p.T(), dir.Mkdir())
	isDir, err := dir.IsDir()
	require.NoError(p.T(), err)
	p.True(isDir)
}

func (p *PathSuite) TestIsntDir() {
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte("hello world!")))
	isDir, err := file.IsDir()
	require.NoError(p.T(), err)
	p.False(isDir)
}

func (p *PathSuite) TestGetLatest() {
	now := time.Now()
	for i := 0; i < 5; i++ {
		file := p.tmpdir.Join(fmt.Sprintf("file%d.txt", i))
		require.NoError(p.T(), file.WriteFile([]byte(fmt.Sprintf("hello %d", i))))
		require.NoError(p.T(), file.Chtimes(now, now))
		now = now.Add(time.Duration(1) * time.Hour)
	}

	latest, err := p.tmpdir.GetLatest()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), "file4.txt", latest.Name())
}

func (p *PathSuite) TestGetLatestEmpty() {
	latest, err := p.tmpdir.GetLatest()
	require.NoError(p.T(), err)
	assert.Nil(p.T(), latest)
}

func (p *PathSuite) TestOpen() {
	msg := "cubs > cardinals"
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte(msg)))
	fileHandle, err := file.Open()
	require.NoError(p.T(), err)

	readBytes := make([]byte, len(msg)+5)
	n, err := fileHandle.Read(readBytes)
	assert.NoError(p.T(), err)
	p.Equal(len(msg), n)
	p.Equal(msg, string(readBytes[0:n]))
}

func (p *PathSuite) TestOpenFile() {
	file := p.tmpdir.Join("file.txt")
	fileHandle, err := file.OpenFile(os.O_RDWR | os.O_CREATE)
	require.NoError(p.T(), err)

	msg := "do you play croquet?"
	n, err := fileHandle.WriteString(msg)
	p.Equal(len(msg), n)
	p.NoError(err)

	bytes := make([]byte, len(msg)+5)
	n, err = fileHandle.ReadAt(bytes, 0)
	p.Equal(len(msg), n)
	p.True(errors.Is(err, io.EOF))
	p.Equal(msg, string(bytes[0:n]))
}

func (p *PathSuite) TestDirExists() {
	dir1 := p.tmpdir.Join("subdir")
	exists, err := dir1.DirExists()
	require.NoError(p.T(), err)
	p.False(exists)

	require.NoError(p.T(), dir1.Mkdir())
	exists, err = dir1.DirExists()
	require.NoError(p.T(), err)
	p.True(exists)
}

func (p *PathSuite) TestIsFile() {
	file1 := p.tmpdir.Join("file.txt")

	require.NoError(p.T(), file1.WriteFile([]byte("")))
	exists, err := file1.IsFile()
	require.NoError(p.T(), err)
	p.True(exists)
}

func (p *PathSuite) TestIsEmpty() {
	file1 := p.tmpdir.Join("file.txt")

	require.NoError(p.T(), file1.WriteFile([]byte("")))
	isEmpty, err := file1.IsEmpty()
	require.NoError(p.T(), err)
	p.True(isEmpty)
}

func (p *PathSuite) TestIsSymlink() {
	file1 := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file1.WriteFile([]byte("")))

	symlink := p.tmpdir.Join("symlink")
	p.NoError(symlink.Symlink(file1))
	isSymlink, err := symlink.IsSymlink()
	require.NoError(p.T(), err)
	p.True(isSymlink)

	stat, _ := symlink.Stat()
	p.T().Logf("%v", stat.Mode())
	p.T().Logf(symlink.String())
}

func (p *PathSuite) TestResolveAll() {
	home := p.tmpdir.Join("mnt", "nfs", "data", "users", "home", "LandonTClipp")
	require.NoError(p.T(), home.MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks").MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks", "home").Symlink(NewPath("../data/users/home")))
	require.NoError(p.T(), p.tmpdir.Join("home").Symlink(NewPath("./mnt/nfs/symlinks/home")))

	resolved, err := p.tmpdir.Join("home/LandonTClipp").ResolveAll()
	p.T().Log(resolved.String())
	require.NoError(p.T(), err)

	homeResolved, err := home.ResolveAll()
	require.NoError(p.T(), err)

	p.Equal(homeResolved.Clean().String(), resolved.Clean().String())
}

func (p *PathSuite) TestResolveAllAbsolute() {
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "data", "users", "home", "LandonTClipp").MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks").MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks", "home").Symlink(p.tmpdir.Join("mnt", "nfs", "data", "users", "home")))
	require.NoError(p.T(), p.tmpdir.Join("home").Symlink(NewPath("./mnt/nfs/symlinks/home")))

	resolved, err := p.tmpdir.Join("home", "LandonTClipp").ResolveAll()
	p.NoError(err)
	resolvedParts := resolved.Parts()
	p.Equal(
		strings.Join(
			[]string{"mnt", "nfs", "data", "users", "home", "LandonTClipp"}, resolved.Sep,
		),
		strings.Join(resolvedParts[len(resolvedParts)-6:], resolved.Sep))
}

func (p *PathSuite) TestEquals() {
	hello1 := p.tmpdir.Join("hello", "world")
	require.NoError(p.T(), hello1.MkdirAll())
	hello2 := p.tmpdir.Join("hello", "world")
	require.NoError(p.T(), hello2.MkdirAll())

	p.True(hello1.Equals(hello2))
}

func (p *PathSuite) TestDeepEquals() {
	hello := p.tmpdir.Join("hello.txt")
	require.NoError(p.T(), hello.WriteFile([]byte("hello")))
	symlink := p.tmpdir.Join("symlink")
	require.NoError(p.T(), symlink.Symlink(hello))

	equals, err := hello.DeepEquals(symlink)
	p.NoError(err)
	p.True(equals)
}

func (p *PathSuite) TestReadDir() {
	require.NoError(p.T(), TwoFilesAtRootTwoInSubdir(p.tmpdir))
	paths, err := p.tmpdir.ReadDir()
	p.NoError(err)
	p.Equal(3, len(paths))
}

func (p *PathSuite) TestReadDirInvalidString() {
	paths, err := p.tmpdir.Join("i_dont_exist").ReadDir()
	p.Error(err)
	p.Equal(0, len(paths))
}

func (p *PathSuite) TestCreate() {
	msg := "hello world"
	file, err := p.tmpdir.Join("hello.txt").Create()
	p.NoError(err)
	defer file.Close()
	n, err := file.WriteString(msg)
	p.Equal(len(msg), n)
	p.NoError(err)
}

func (p *PathSuite) TestGlobFunction() {
	hello1 := p.tmpdir.Join("hello1.txt")
	require.NoError(p.T(), hello1.WriteFile([]byte("hello")))

	hello2 := p.tmpdir.Join("hello2.txt")
	require.NoError(p.T(), hello2.WriteFile([]byte("hello2")))

	paths, err := Glob(p.tmpdir.Fs(), p.tmpdir.Join("hello1*").String())
	p.NoError(err)
	require.Equal(p.T(), 1, len(paths))
	p.True(hello1.Equals(paths[0]), "received an unexpected path: %v", paths[0])
}

func TestPathSuite(t *testing.T) {
	suite.Run(t, new(PathSuite))
}

func TestPath_IsAbsolute(t *testing.T) {
	type fields struct {
		path string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"root path", fields{"/"}, true},
		{"absolute path", fields{"./"}, false},
		{"absolute path", fields{"."}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Path{
				path: tt.fields.path,
			}
			if got := p.IsAbsolute(); got != tt.want {
				t.Errorf("Path.IsAbsolute() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Join(t *testing.T) {
	type args struct {
		elems []string
	}
	tests := []struct {
		name   string
		fields string
		args   args
		want   string
	}{
		{"join absolute root", "/", args{[]string{"foo", "bar"}}, "/foo/bar"},
		{"join relative root", "./", args{[]string{"foo", "bar"}}, "foo/bar"},
		{"join with existing path", "./foo", args{[]string{"bar", "baz"}}, "foo/bar/baz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := afero.NewMemMapFs()
			p := NewPathAfero(tt.fields, a)
			want := NewPathAfero(tt.want, a)
			if got := p.Join(tt.args.elems...).Clean(); !reflect.DeepEqual(got, want) {
				t.Errorf("Path.Join() = %v, want %v", got, want)
			}
		})
	}
}

func TestPath_Parent(t *testing.T) {
	type fields struct {
		path            string
		fs              afero.Fs
		DefaultFileMode os.FileMode
	}
	tests := []struct {
		name   string
		fields string
		want   string
	}{
		{"absolute path", "/path/to/foo.txt", "/path/to"},
		{"relative path", "foo.txt", "."},
		{"root of relative", ".", "."},
		{"root of relative with slash", "./", "."},
		{"absolute root", "/", "/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := afero.NewMemMapFs()
			p := NewPathAfero(tt.fields, a)
			want := NewPathAfero(tt.want, a)
			if got := p.Parent(); !reflect.DeepEqual(got, want) {
				t.Errorf("Path.Parent() = %v, want %v", got, want)
			}
		})
	}
}

func TestPathPosix_RelativeTo(t *testing.T) {
	a := afero.NewMemMapFs()
	type fields struct {
		path            string
		fs              afero.Fs
		DefaultFileMode os.FileMode
	}
	tests := []struct {
		name      string
		fieldPath string
		args      string
		want      string
		wantErr   bool
	}{
		{"0", "/etc/passwd", "/", "etc/passwd", false},
		{"1", "/etc/passwd", "/etc", "passwd", false},
		{"2", "/etc/passwd/", "/etc", "passwd", false},
		{"3", "/etc/passwd", "/etc/", "passwd", false},
		{"4", "/etc/passwd/", "/etc/", "passwd", false},
		{"5", "/etc/passwd/", "/usr/", "/etc/passwd/", true},
		{"6", "/", "/", ".", false},
		{"7", "./foo/bar", "foo", "bar", false},
		{"8", "/a/b/c/d/file.txt", "/a/b/c/d/", "file.txt", false},
		{"9", "/cool/cats/write/cool/code/file.csv", "/cool/cats/write", "cool/code/file.csv", false},
		{"10", "/etc/passwd", "////////////", "etc/passwd", false},
		{"11", "/etc/passwd/////", "/", "etc/passwd", false},
		{"12", "/etc/passwd", "/etc/passwd/test", "/etc/passwd", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPathAfero(tt.fieldPath, a)
			got, err := p.RelativeTo(NewPathAfero(tt.args, a))
			if (err != nil) != tt.wantErr {
				t.Errorf("Path.RelativeTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, NewPathAfero(tt.want, a)) {
				t.Errorf("Path.RelativeTo() = %v, want %v", got, tt.want)
			}
		})
		a = afero.NewMemMapFs()
	}
}

func TestPath_Parts(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{"0", "/path/to/thingy", []string{"/", "path", "to", "thingy"}},
		{"1", "path/to/thingy", []string{"path", "to", "thingy"}},
		{"2", "/", []string{"/"}},
		{"3", "./path/to/thingy", []string{"path", "to", "thingy"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPathAfero(tt.path, afero.NewMemMapFs())
			if got := p.Parts(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Path.Parts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_Copy(t *testing.T) {
	tests := []struct {
		name            string
		srcContents     string
		dstContents     string
		wantDstContents string
		createDstFile   bool
		wantErr         bool
	}{
		{
			name:            "copy empty file to existing non-empty file",
			srcContents:     "",
			dstContents:     "foobar",
			wantDstContents: "",
			createDstFile:   true,
		},
		{
			name:            "copy empty file to existing empty file",
			srcContents:     "",
			dstContents:     "",
			wantDstContents: "",
			createDstFile:   true,
		},
		{
			name:            "copy empty file to non-existing file",
			srcContents:     "",
			wantDstContents: "",
			createDstFile:   false,
		},
		{
			name:            "copy non-empty file to existing non-empty file",
			srcContents:     "foobar",
			dstContents:     "hello world",
			wantDstContents: "foobar",
			createDstFile:   true,
		},
		{
			name:            "copy non-empty file to existing empty file",
			srcContents:     "foobar",
			dstContents:     "",
			wantDstContents: "foobar",
			createDstFile:   true,
		},
		{
			name:            "copy non-empty file to non-existing file",
			srcContents:     "foobar",
			wantDstContents: "foobar",
			createDstFile:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := NewPath(t.TempDir())
			src := tmpdir.Join("src.txt")
			dst := tmpdir.Join("dst.txt")
			require.NoError(t, src.WriteFile([]byte(tt.srcContents)))

			if tt.createDstFile {
				require.NoError(t, dst.WriteFile([]byte(tt.dstContents)))
			}

			_, err := src.Copy(dst)
			if !tt.wantErr {
				require.NoError(t, err)
			}

			dstBytes, err := dst.ReadFile()
			require.NoError(t, err)
			assert.Equal(t, []byte(tt.wantDstContents), dstBytes)
		})
	}
}
