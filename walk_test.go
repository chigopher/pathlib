package pathlib

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// WalkSuiteAll is a set of tests that should be run
// for all walk algorithms. It asserts the behaviors that
// are identical between all algorithms.
type WalkSuiteAll struct {
	suite.Suite
	walk      *Walk
	root      *Path
	algorithm Algorithm
	Fs        afero.Fs
}

func (w *WalkSuiteAll) SetupTest() {
	var err error

	tmpdir, err := ioutil.TempDir("", "")
	require.NoError(w.T(), err)

	w.Fs = afero.NewOsFs()
	w.root = NewPathAfero(tmpdir, w.Fs)
	w.walk, err = NewWalk(w.root)
	require.NoError(w.T(), err)
	w.walk.Opts.Algorithm = w.algorithm
}

func (w *WalkSuiteAll) TeardownTest() {
	require.NoError(w.T(), w.root.RemoveAll())
}

func (w *WalkSuiteAll) TestHello() {
	require.NoError(w.T(), HelloWorld(w.root))

	walkFunc := MockWalkFunc{}
	walkFunc.On("Execute", mock.Anything, mock.Anything, nil).Return(nil)
	w.NoError(w.walk.Walk(walkFunc.Execute))
	walkFunc.AssertExpectations(w.T())
}

func (w *WalkSuiteAll) TestTwoFiles() {
	require.NoError(w.T(), NFiles(w.root, 2))

	walkFunc := MockWalkFunc{}
	walkFunc.On("Execute", mock.Anything, mock.Anything, nil).Return(nil)
	w.NoError(w.walk.Walk(walkFunc.Execute))
	walkFunc.AssertExpectations(w.T())
	walkFunc.AssertNumberOfCalls(w.T(), "Execute", 2)
}

func (w *WalkSuiteAll) TestTwoFilesNested() {
	require.NoError(w.T(), TwoFilesAtRootTwoInSubdir(w.root))

	walkFunc := MockWalkFunc{}
	walkFunc.On("Execute", mock.Anything, mock.Anything, nil).Return(nil)
	w.NoError(w.walk.Walk(walkFunc.Execute))
	walkFunc.AssertExpectations(w.T())
	walkFunc.AssertNumberOfCalls(w.T(), "Execute", 5)
}

func (w *WalkSuiteAll) TestZeroDepth() {
	w.walk.Opts.Depth = 0
	w.walk.Opts.FollowSymlinks = true
	require.NoError(w.T(), TwoFilesAtRootTwoInSubdir(w.root))

	walkFunc := MockWalkFunc{}
	walkFunc.On("Execute", mock.Anything, mock.Anything, nil).Return(nil)
	w.NoError(w.walk.Walk(walkFunc.Execute))
	walkFunc.AssertExpectations(w.T())

	// WalkFunc should be called three times because there are two files and
	// one subdir.
	walkFunc.AssertNumberOfCalls(w.T(), "Execute", 3)
}

func (w *WalkSuiteAll) TestStopWalk() {
	require.NoError(w.T(), TwoFilesAtRootTwoInSubdir(w.root))

	walkFunc := MockWalkFunc{}
	walkFunc.On("Execute", mock.Anything, mock.Anything, nil).Return(ErrStopWalk)
	w.NoError(w.walk.Walk(walkFunc.Execute))
	walkFunc.AssertExpectations(w.T())
	walkFunc.AssertNumberOfCalls(w.T(), "Execute", 1)
}

func (w *WalkSuiteAll) TestWalkFuncErr() {
	require.NoError(w.T(), TwoFilesAtRootTwoInSubdir(w.root))

	wantErr := fmt.Errorf("Aww shoot")
	walkFunc := MockWalkFunc{}
	walkFunc.On("Execute", mock.Anything, mock.Anything, nil).Return(wantErr)
	w.EqualError(w.walk.Walk(walkFunc.Execute), wantErr.Error(), "did not receive the expected err")
	walkFunc.AssertExpectations(w.T())
	walkFunc.AssertNumberOfCalls(w.T(), "Execute", 1)
}

func (w *WalkSuiteAll) TestPassesQuerySpecification() {
	file := w.root.Join("file.txt")
	require.NoError(w.T(), file.WriteFile([]byte("hello")))

	stat, err := file.Stat()
	require.NoError(w.T(), err)

	// File tests
	w.walk.Opts.VisitFiles = false
	passes, err := w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.False(passes, "specified to not visit files, but passed anyway")

	w.walk.Opts.VisitFiles = true
	passes, err = w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.True(passes, "specified to visit files, but didn't pass")

	w.walk.Opts.MinimumFileSize = 100
	passes, err = w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.False(passes, "specified large file size, but passed anyway")

	w.walk.Opts.MinimumFileSize = 0
	passes, err = w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.True(passes, "specified smallfile size, but didn't pass")

	// Directory tests
	dir := w.root.Join("subdir")
	require.NoError(w.T(), dir.MkdirAll())

	stat, err = dir.Stat()
	require.NoError(w.T(), err)

	w.walk.Opts.VisitDirs = false
	passes, err = w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.False(passes, "specified to not visit directories, but passed anyway")

	w.walk.Opts.VisitDirs = true
	passes, err = w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.True(passes, "specified to visit directories, but didn't pass")

	// Symlink tests
	symlink := w.root.Join("symlink")
	require.NoError(w.T(), symlink.Symlink(file))

	stat, err = symlink.Lstat()
	require.NoError(w.T(), err)

	w.walk.Opts.VisitSymlinks = false
	passes, err = w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.False(passes, "specified to not visit symlinks, but passed anyway")

	w.walk.Opts.VisitSymlinks = true
	passes, err = w.walk.passesQuerySpecification(stat)
	require.NoError(w.T(), err)
	w.True(passes, "specified to visit symlinks, but didn't pass")
}

func TestWalkSuite(t *testing.T) {
	for _, algorithm := range []Algorithm{
		AlgorithmBasic,
		AlgorithmDepthFirst,
	} {
		walkSuite := new(WalkSuiteAll)
		walkSuite.algorithm = algorithm
		suite.Run(t, walkSuite)
	}
}

func TestDefaultWalkOpts(t *testing.T) {
	tests := []struct {
		name string
		want *WalkOpts
	}{
		{"assert defaults", &WalkOpts{
			Depth:           -1,
			Algorithm:       AlgorithmBasic,
			FollowSymlinks:  false,
			MinimumFileSize: -1,
			MaximumFileSize: -1,
			VisitFiles:      true,
			VisitDirs:       true,
			VisitSymlinks:   true,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultWalkOpts(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultWalkOpts() = %v, want %v", got, tt.want)
			}
		})
	}
}

var ConfusedWandering Algorithm = 0xBADC0DE

func TestWalk_Walk(t *testing.T) {
	type fields struct {
		Opts *WalkOpts
		root *Path
	}
	type args struct {
		walkFn WalkFunc
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Bad algoritm",
			fields: fields{
				Opts: &WalkOpts{Algorithm: ConfusedWandering},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Walk{
				Opts: tt.fields.Opts,
				root: tt.fields.root,
			}
			if err := w.Walk(tt.args.walkFn); (err != nil) != tt.wantErr {
				t.Errorf("Walk.Walk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewWalk(t *testing.T) {
	type args struct {
		opts []WalkOptsFunc
	}
	tests := []struct {
		name    string
		args    args
		want    *Walk
		wantErr bool
	}{
		{
			name: "test all WalkOptsFunc",
			args: args{
				opts: []WalkOptsFunc{
					WalkVisitSymlinks(true),
					WalkVisitDirs(true),
					WalkVisitFiles(true),
					WalkMaximumFileSize(1000),
					WalkMinimumFileSize(500),
					WalkFollowSymlinks(true),
					WalkAlgorithm(AlgorithmDepthFirst),
					WalkDepth(10),
				},
			},
			want: &Walk{
				Opts: &WalkOpts{
					VisitSymlinks:   true,
					VisitDirs:       true,
					VisitFiles:      true,
					MaximumFileSize: 1000,
					MinimumFileSize: 500,
					FollowSymlinks:  true,
					Algorithm:       AlgorithmDepthFirst,
					Depth:           10,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := NewPath(t.TempDir())
			got, err := NewWalk(tmpdir, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWalk() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.want.root = tmpdir
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWalk() = %v, want %v", got, tt.want)
			}
		})
	}
}
