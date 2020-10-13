package pathlib

import "fmt"

// The following functions provide different "scenarios"
// that you might encounter in a filesystem tree.

func HelloWorld(root *Path) error {
	hello := root.Join("hello.txt")
	return hello.WriteFile([]byte("hello world"), 0o644)
}

func OneFile(root *Path, name string, content string) error {
	file := root.Join(name)
	return file.WriteFile([]byte(content), 0o644)
}

func NFiles(root *Path, n int) error {
	for i := 0; i < n; i++ {
		if err := OneFile(root, fmt.Sprintf("file%d.txt", i), fmt.Sprintf("file%d contents", i)); err != nil {
			return err
		}
	}
	return nil
}

// TwoFilesAtRootTwoInSubdir creates two files in the root dir,
// a directory, and creates two files inside that new directory.
func TwoFilesAtRootTwoInSubdir(root *Path) error {
	if err := NFiles(root, 2); err != nil {
		return err
	}
	subdir := root.Join("subdir")
	if err := subdir.Mkdir(); err != nil {
		return err
	}
	return NFiles(subdir, 2)
}
