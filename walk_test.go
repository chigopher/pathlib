package pathlib

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/LandonTClipp/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// *********
// * TESTS *
// *********

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

	w.Fs = afero.NewMemMapFs()
	w.root = NewPathAfero("/", w.Fs)
	w.walk, err = NewWalk(w.root)
	require.NoError(w.T(), err)
	w.walk.Opts.Algorithm = w.algorithm
}

func (w *WalkSuiteAll) TeardownTest() {

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
