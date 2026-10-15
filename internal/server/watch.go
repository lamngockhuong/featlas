package server

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debounce batches file events before a rebuild.
const Debounce = 300 * time.Millisecond

// skipDirs are never watched inside source directories.
var skipDirs = map[string]bool{".git": true, "node_modules": true}

// watchSet is what one snapshot asks the watcher to observe.
type watchSet struct {
	dirs       map[string]bool // directories to register with fsnotify
	contentDir string
	files      map[string]bool // source files
	trees      []string        // source directories (prefixes)
}

func (s *Server) watchSet() watchSet {
	snap := s.Snapshot()
	ws := watchSet{dirs: map[string]bool{}, contentDir: snap.Index.Dir, files: map[string]bool{}}
	addTree(ws.dirs, snap.Index.Dir)
	for _, e := range snap.Index.All() {
		for _, src := range e.Sources {
			abs, err := snap.Index.SourcePath(src)
			if err != nil {
				continue
			}
			if strings.HasSuffix(src, "/") {
				ws.trees = append(ws.trees, abs+string(filepath.Separator))
				addTree(ws.dirs, abs)
				continue
			}
			ws.files[abs] = true
			// Watch the parent so a file deleted and recreated (as editors
			// do on save) keeps being observed.
			if st, err := os.Stat(filepath.Dir(abs)); err == nil && st.IsDir() {
				ws.dirs[filepath.Dir(abs)] = true
			}
		}
	}
	return ws
}

func addTree(dirs map[string]bool, root string) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if p != root && (skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".")) {
			return filepath.SkipDir
		}
		dirs[p] = true
		return nil
	})
}

// relevant reports whether a changed path can affect the index.
func (ws watchSet) relevant(path string) bool {
	if strings.HasPrefix(path, ws.contentDir+string(filepath.Separator)) || ws.files[path] {
		return true
	}
	for _, t := range ws.trees {
		if strings.HasPrefix(path, t) {
			return true
		}
	}
	return false
}

// Watch rebuilds the index when content or source files change, until ctx
// ends. Watched directories follow the sources of the latest snapshot.
func (s *Server) Watch(ctx context.Context) error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	watched := map[string]bool{}
	ws := s.sync(w, watched)
	var timer *time.Timer
	var fire <-chan time.Time
	for {
		select {
		case <-ctx.Done():
			return nil
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			s.log.Warn("file watcher", "error", err)
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			if !ws.relevant(ev.Name) && (!ev.Has(fsnotify.Create) || !isDir(ev.Name)) {
				continue
			}
			if timer == nil {
				timer = time.NewTimer(Debounce)
			} else {
				timer.Reset(Debounce)
			}
			fire = timer.C
		case <-fire:
			fire = nil
			if err := s.Rebuild(); err != nil {
				s.log.Error("rebuild failed", "error", err)
				continue
			}
			ws = s.sync(w, watched)
		}
	}
}

// sync registers new directories and drops ones no longer needed.
func (s *Server) sync(w *fsnotify.Watcher, watched map[string]bool) watchSet {
	ws := s.watchSet()
	for d := range ws.dirs {
		if !watched[d] {
			if err := w.Add(d); err != nil {
				s.log.Warn("cannot watch directory", "dir", d, "error", err)
				continue
			}
			watched[d] = true
		}
	}
	for d := range watched {
		if !ws.dirs[d] {
			_ = w.Remove(d)
			delete(watched, d)
		}
	}
	return ws
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
