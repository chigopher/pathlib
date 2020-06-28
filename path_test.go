package pathlib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
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

	linkLocation, err := symlink.Resolve()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), p.tmpdir.path, linkLocation.path)
}

func (p *PathSuite) TestSymlinkBadFs() {
	symlink := p.tmpdir.Join("symlink")
	symlink.afero.Fs = afero.NewMemMapFs()

	assert.Error(p.T(), symlink.Symlink(p.tmpdir))
}

func (p *PathSuite) TestJoin() {
	joined := p.tmpdir.Join("test1")
	assert.Equal(p.T(), filepath.Join(p.tmpdir.Path(), "test1"), joined.Path())
}

func (p *PathSuite) TestWriteAndRead() {
	expectedBytes := []byte("hello world!")
	file := p.tmpdir.Join("test.txt")
	require.NoError(p.T(), file.WriteFile(expectedBytes, 0o755))
	bytes, err := file.ReadFile()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), expectedBytes, bytes)
}

func (p *PathSuite) TestChmod() {
	file := p.tmpdir.Join("file1.txt")
	require.NoError(p.T(), file.WriteFile([]byte(""), 0o755))

	file.Chmod(0o777)
	fileInfo, err := file.Stat()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), os.FileMode(0o777), fileInfo.Mode()&os.ModePerm)

	file.Chmod(0o755)
	fileInfo, err = file.Stat()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), os.FileMode(0o755), fileInfo.Mode()&os.ModePerm)
}

func (p *PathSuite) TestMkdir() {
	subdir := p.tmpdir.Join("subdir")
	assert.NoError(p.T(), subdir.Mkdir(0o777))
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestMkdirParentsDontExist() {
	subdir := p.tmpdir.Join("subdir1", "subdir2")
	assert.Error(p.T(), subdir.Mkdir(0o777))
}

func (p *PathSuite) TestMkdirAll() {
	subdir := p.tmpdir.Join("subdir")
	assert.NoError(p.T(), subdir.MkdirAll(0o777))
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestMkdirAllMultipleSubdirs() {
	subdir := p.tmpdir.Join("subdir1", "subdir2", "subdir3")
	assert.NoError(p.T(), subdir.MkdirAll(0o777))
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestRenamePath() {
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte("hello world!"), 0o755))

	newPath := p.tmpdir.Join("file2.txt")

	newFile, err := file.RenamePath(newPath)
	assert.NoError(p.T(), err)
	assert.Equal(p.T(), newFile.Name(), "file2.txt")

	newBytes, err := newFile.ReadFile()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), []byte("hello world!"), newBytes)

	oldFileExists, err := file.Exists()
	require.NoError(p.T(), err)
	assert.False(p.T(), oldFileExists)
}

func (p *PathSuite) TestGetLatest() {
	now := time.Now()
	for i := 0; i < 5; i++ {
		file := p.tmpdir.Join(fmt.Sprintf("file%d.txt", i))
		require.NoError(p.T(), file.WriteFile([]byte(fmt.Sprintf("hello %d", i)), 0o644))
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

func TestPathSuite(t *testing.T) {
	suite.Run(t, new(PathSuite))
}

func TestPath_Join(t *testing.T) {
	type fields struct {
		path string
	}
	type args struct {
		elems []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Path
	}{
		{"join absolute root", fields{"/"}, args{[]string{"foo", "bar"}}, &Path{"/foo/bar", afero.Afero{}}},
		{"join relative root", fields{"./"}, args{[]string{"foo", "bar"}}, &Path{"foo/bar", afero.Afero{}}},
		{"join with existing path", fields{"./foo"}, args{[]string{"bar", "baz"}}, &Path{"foo/bar/baz", afero.Afero{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Path{
				path: tt.fields.path,
			}
			if got := p.Join(tt.args.elems...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Path.Join() = %v, want %v", got, tt.want)
			}
		})
	}
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

func TestPath_Parent(t *testing.T) {
	type fields struct {
		path  string
		afero afero.Afero
	}
	tests := []struct {
		name   string
		fields fields
		want   *Path
	}{
		{"absolute path", fields{path: "/path/to/foo.txt"}, &Path{"/path/to", afero.Afero{}}},
		{"relative path", fields{path: "foo.txt"}, &Path{".", afero.Afero{}}},
		{"root of relative", fields{path: "."}, &Path{".", afero.Afero{}}},
		{"root of relative with slash", fields{path: "./"}, &Path{".", afero.Afero{}}},
		{"absolute root", fields{path: "/"}, &Path{"/", afero.Afero{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Path{
				path:  tt.fields.path,
				afero: tt.fields.afero,
			}
			if got := p.Parent(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Path.Parent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPath_RelativeTo(t *testing.T) {
	a := afero.NewMemMapFs()
	type fields struct {
		path  string
		afero afero.Afero
	}
	type args struct {
		other *Path
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Path
		wantErr bool
	}{
		{"1", fields{"/etc/passwd", afero.Afero{a}}, args{NewPathAfero("/", a)}, NewPathAfero("etc/passwd", a), false},
		{"2", fields{"/etc/passwd", afero.Afero{a}}, args{NewPathAfero("/etc", a)}, NewPathAfero("passwd", a), false},
		{"3", fields{"/etc/passwd/", afero.Afero{a}}, args{NewPathAfero("/etc", a)}, NewPathAfero("passwd", a), false},
		{"4", fields{"/etc/passwd", afero.Afero{a}}, args{NewPathAfero("/etc/", a)}, NewPathAfero("passwd", a), false},
		{"5", fields{"/etc/passwd/", afero.Afero{a}}, args{NewPathAfero("/etc/", a)}, NewPathAfero("passwd", a), false},
		{"6", fields{"/etc/passwd/", afero.Afero{a}}, args{NewPathAfero("/usr/", a)}, nil, true},
		{"7", fields{"/", afero.Afero{a}}, args{NewPathAfero("/", a)}, NewPathAfero(".", a), false},
		{"8", fields{"./foo/bar", afero.Afero{a}}, args{NewPathAfero("foo", a)}, NewPathAfero("bar", a), false},
		{"9", fields{"/a/b/c/d/file.txt", afero.Afero{a}}, args{NewPathAfero("/a/b/c/d/", a)}, NewPathAfero("file.txt", a), false},
		{"10", fields{"/cool/cats/write/cool/code/file.csv", afero.Afero{a}}, args{NewPathAfero("/cool/cats/write", a)}, NewPathAfero("cool/code/file.csv", a), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Path{
				path:  tt.fields.path,
				afero: tt.fields.afero,
			}
			got, err := p.RelativeTo(tt.args.other)
			if (err != nil) != tt.wantErr {
				t.Errorf("Path.RelativeTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Path.RelativeTo() = %v, want %v", got, tt.want)
			}
		})
		a = afero.NewMemMapFs()
	}
}
