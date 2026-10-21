package settings

import (
	"io"
	"os"
	"path/filepath"

	"github.com/yetone/magpie/internal/appdir"
)

// migrated is what runs after the files have been carried over, for a
// build that has something of its own to say about them. It is a var so
// that a build tagged for another product can set it, and magpie's own
// builds leave it doing nothing.
var migrated = func() {}

// Migrate carries the files of an install that predates the name over:
// ~/.config/magpie and ~/.cache/magpie — and, for one that never ran
// magpie, ~/.config/dial and ~/.cache/dial — become ~/.config/echo and
// ~/.cache/echo (the app is Echo). Only the newest name's folder copies:
// a magpie's files said dial's over as its own migration ran, so copying
// magpie's when they are both kept carries the newer state. It copies
// rather than moves, so a magpie that is still running keeps working,
// and it only fills folders that do not exist yet. The old folders can
// be deleted once nothing uses them. A portable app has only its own
// data folder and copies nothing in.
func Migrate() {
	if Portable() != "" {
		return
	}
	dir := filepath.Dir(Dir())
	src := filepath.Join(dir, "magpie")
	if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
		src = filepath.Join(dir, "dial")
	}
	copyTree(src, Dir())
	cache := appdir.Getenv("XDG_CACHE_HOME")
	if cache == "" {
		home, _ := os.UserHomeDir()
		cache = filepath.Join(home, ".cache")
	}
	csrc := filepath.Join(cache, "magpie")
	if fi, err := os.Stat(csrc); err != nil || !fi.IsDir() {
		csrc = filepath.Join(cache, "dial")
	}
	copyTree(csrc, filepath.Join(cache, "echo"))
	// the OS's own cache folder, ~/Library/Caches on a Mac,
	// %LocalAppData% on Windows, named magpie before this one
	if sc, err := appdir.SystemCache(); err == nil {
		copyTree(filepath.Join(filepath.Dir(sc), "magpie"), sc)
	}
	migrated()
}

// copyTree copies src into dst, files and modes alike, when src exists and
// dst does not. Errors are not reported: the worst case is a fresh start,
// which is what a missing folder means anyway.
func copyTree(src, dst string) {
	if _, err := os.Stat(dst); err == nil {
		return
	}
	if fi, err := os.Stat(src); err != nil || !fi.IsDir() {
		return
	}
	_ = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = os.MkdirAll(out, info.Mode().Perm())
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		in, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer in.Close()
		f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return nil
		}
		defer f.Close()
		_, _ = io.Copy(f, in)
		return nil
	})
}
