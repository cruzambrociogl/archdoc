package store

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Tree writes a repository as it was at a revision into a fresh directory, and returns it with the
// function that removes it. The directory is named as the repository is, because a model takes its
// name from where it was read.
//
// This is the one place archdoc runs git. Everything else reads the two files that name the
// checked-out commit (render.Commit), so generate and serve work where git is not installed; but
// the files of another commit live in git's object store, packed and delta-compressed, and reading
// that is git's job. Comparing two commits therefore needs git; nothing else does.
func Tree(root, rev string) (dir string, remove func(), err error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}
	if strings.HasPrefix(rev, "-") {
		return "", nil, fmt.Errorf("%q is not a revision", rev)
	}
	var tarball, problem bytes.Buffer
	cmd := exec.Command("git", "-C", abs, "archive", "--format=tar", rev)
	cmd.Stdout, cmd.Stderr = &tarball, &problem
	if err := cmd.Run(); err != nil {
		var missing *exec.Error
		if errors.As(err, &missing) {
			return "", nil, errors.New("comparing commits needs git, which was not found")
		}
		return "", nil, fmt.Errorf("git could not read %s: %s", rev, strings.TrimSpace(problem.String()))
	}

	tmp, err := os.MkdirTemp("", "archdoc-tree-")
	if err != nil {
		return "", nil, err
	}
	remove = func() { os.RemoveAll(tmp) }
	dir = filepath.Join(tmp, filepath.Base(abs))
	if err := os.Mkdir(dir, 0o755); err != nil {
		remove()
		return "", nil, err
	}
	tr := tar.NewReader(&tarball)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			remove()
			return "", nil, err
		}
		// Only what is under the directory, and only files and directories: a link is not followed.
		target := filepath.Join(dir, filepath.FromSlash(h.Name))
		if !strings.HasPrefix(target, dir+string(filepath.Separator)) && target != dir {
			continue
		}
		switch h.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			if err = os.MkdirAll(filepath.Dir(target), 0o755); err == nil {
				var f *os.File
				if f, err = os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
					_, err = io.Copy(f, tr)
					f.Close()
				}
			}
		}
		if err != nil {
			remove()
			return "", nil, err
		}
	}
	return dir, remove, nil
}
