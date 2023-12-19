package pathlib

import (
	"fmt"
	os "os"
	"reflect"
	"slices"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

	tmpdir, err := os.MkdirTemp("", "")
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

type FSObject struct {
	path     *Path
	contents string
	dir      bool
}

func TestWalkerOrder(t *testing.T) {
	type test struct {
		name          string
		algorithm     Algorithm
		walkOpts      []WalkOptsFunc
		objects       []FSObject
		expectedOrder []*Path
	}
	for _, tt := range []test{
		{
			name:      "Pre-Order DFS simple",
			algorithm: AlgorithmPreOrderDepthFirst,
			objects: []FSObject{
				{path: NewPath("1.txt")},
				{path: NewPath("2.txt")},
				{path: NewPath("3.txt")},
				{path: NewPath("subdir"), dir: true},
				{path: NewPath("subdir").Join("4.txt")},
			},
			walkOpts: []WalkOptsFunc{WalkVisitDirs(true)},
			expectedOrder: []*Path{
				NewPath("1.txt"),
				NewPath("2.txt"),
				NewPath("3.txt"),
				NewPath("subdir"),
				NewPath("subdir").Join("4.txt"),
			},
		},
		{
			name:      "Post-Order DFS simple",
			algorithm: AlgorithmDepthFirst,
			objects: []FSObject{
				{path: NewPath("1.txt")},
				{path: NewPath("2.txt")},
				{path: NewPath("3.txt")},
				{path: NewPath("subdir"), dir: true},
				{path: NewPath("subdir").Join("4.txt")},
			},
			walkOpts: []WalkOptsFunc{WalkVisitDirs(true)},
			expectedOrder: []*Path{
				NewPath("subdir").Join("4.txt"),
				NewPath("1.txt"),
				NewPath("2.txt"),
				NewPath("3.txt"),
				NewPath("subdir"),
			},
		},
		{
			name:      "Basic simple",
			algorithm: AlgorithmBasic,
			objects: []FSObject{
				{path: NewPath("1")},
				{path: NewPath("2"), dir: true},
				{path: NewPath("2").Join("3")},
				{path: NewPath("4")},
			},
			walkOpts: []WalkOptsFunc{WalkVisitDirs(true)},
			expectedOrder: []*Path{
				NewPath("1"),
				NewPath("2").Join("3"),
				NewPath("2"),
				NewPath("4"),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := NewPath(t.TempDir())
			for _, child := range tt.objects {
				c := root.JoinPath(child.path)
				if child.dir {
					require.NoError(t, c.Mkdir())
					continue
				}
				require.NoError(t, c.WriteFile([]byte(child.contents)))
			}
			opts := []WalkOptsFunc{WalkAlgorithm(tt.algorithm), WalkSortChildren(true)}
			opts = append(opts, tt.walkOpts...)
			walker, err := NewWalk(root, opts...)
			require.NoError(t, err)

			actualOrder := []*Path{}
			require.NoError(
				t,
				walker.Walk(func(path *Path, info os.FileInfo, err error) error {
					require.NoError(t, err)
					relative, err := path.RelativeTo(root)
					require.NoError(t, err)
					actualOrder = append(actualOrder, relative)
					return nil
				}),
			)
			require.Equal(t, len(tt.expectedOrder), len(actualOrder))
			for i, path := range tt.expectedOrder {
				assert.True(t, path.Equals(actualOrder[i]), "incorrect ordering at %d: %s != %s", i, path, actualOrder[i])
			}
		})
	}
}

// TestErrWalkSkipSubtree tests the behavior of each algorithm when we tell it to skip a subtree.
func TestErrWalkSkipSubtree(t *testing.T) {
	type test struct {
		name      string
		algorithm Algorithm
		tree      []*Path
		skipAt    *Path
		expected  []*Path
	}

	for _, tt := range []test{
		{
			// In AlgorithmBasic, the ordering in which children/nodes are visited
			// is filesystem and OS dependent. Some filesystems return paths in a lexically-ordered
			// manner, some return them in the order in which they were created. For this test,
			// we tell the walker to order the children before iterating over them. That way,
			// the test will visit "subdir1/subdir2/foo.txt" before "subdir1/subdir2/subdir3/foo.txt",
			// in which case we would tell the walker to skip the subdir3 subtree before it recursed.
			"Basic",
			AlgorithmBasic,
			nil,
			NewPath("subdir1").Join("subdir2", "foo.txt"),
			[]*Path{
				NewPath("foo1.txt"),
				NewPath("subdir1").Join("foo.txt"),
				NewPath("subdir1").Join("subdir2", "foo.txt"),
			},
		},
		{
			"PreOrderDFS",
			AlgorithmPreOrderDepthFirst,
			nil,
			NewPath("subdir1").Join("subdir2", "foo.txt"),
			[]*Path{
				NewPath("foo1.txt"),
				NewPath("subdir1").Join("foo.txt"),
				NewPath("subdir1").Join("subdir2", "foo.txt"),
			},
		},
		{
			"PreOrderDFS skip at root",
			AlgorithmPreOrderDepthFirst,
			nil,
			NewPath("foo1.txt"),
			[]*Path{
				NewPath("foo1.txt"),
			},
		},
		// Note about the PostOrderDFS case. ErrWalkSkipSubtree effectively
		// has no meaning to this algorithm because in this case, the algorithm
		// visits all children before visiting each node. Thus, our WalkFunc has
		// no opportunity to tell it to skip a particular subtree. This test
		// serves to ensure this behavior doesn't change.
		{
			"PostOrderDFS",
			AlgorithmPostOrderDepthFirst,
			nil,
			NewPath("subdir1").Join("subdir2", "foo.txt"),
			[]*Path{
				NewPath("foo1.txt"),
				NewPath("subdir1").Join("foo.txt"),
				NewPath("subdir1").Join("subdir2", "foo.txt"),
				NewPath("subdir1").Join("subdir2", "subdir3", "foo.txt"),
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := NewPath(t.TempDir())
			walker, err := NewWalk(root, WalkAlgorithm(tt.algorithm), WalkVisitDirs(false), WalkVisitFiles(true), WalkSortChildren(true))
			require.NoError(t, err)

			var tree []*Path
			if tt.tree == nil {
				tree = []*Path{
					NewPath("foo1.txt"),
					NewPath("subdir1").Join("foo.txt"),
					NewPath("subdir1").Join("subdir2", "foo.txt"),
					NewPath("subdir1").Join("subdir2", "subdir3", "foo.txt"),
				}
			}
			for _, path := range tree {
				p := root.JoinPath(path)
				require.NoError(t, p.Parent().MkdirAll())
				require.NoError(t, p.WriteFile([]byte("")))
			}

			visited := map[string]struct{}{}
			require.NoError(t, walker.Walk(func(path *Path, info os.FileInfo, err error) error {
				t.Logf("visited: %v", path.String())
				require.NoError(t, err)
				rel, err := path.RelativeTo(root)
				require.NoError(t, err)
				visited[rel.String()] = struct{}{}
				if rel.Equals(tt.skipAt) {
					return ErrWalkSkipSubtree
				}
				return nil
			}))
			visitedSorted := []string{}
			for key := range visited {
				visitedSorted = append(visitedSorted, key)
			}
			slices.Sort(visitedSorted)

			expected := []string{}
			for _, path := range tt.expected {
				expected = append(expected, path.String())
			}
			assert.Equal(t, expected, visitedSorted)

		})
	}
}
