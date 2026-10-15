// Package server serves the content API, live reload events, content images
// and the embedded web UI.
package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lamngockhuong/muninn/internal/content"
	"github.com/lamngockhuong/muninn/internal/review"
)

// Options configure the server.
type Options struct {
	Dir      string
	ReadOnly bool
	// ReadOnlyReason explains to the client why reviewing is disabled.
	ReadOnlyReason string
	Token          string // when set, write requests need "Authorization: Bearer <token>"
	Local          bool   // bound to loopback: expose the repo path for vscode:// links
	// Host is the bind address. In local mode only loopback names and this
	// host are accepted in the Host header, which defeats DNS rebinding.
	Host    string
	Version string
	UI      http.Handler
	Logger  *slog.Logger
}

// Server holds the current snapshot and the live reload subscribers.
type Server struct {
	opt    Options
	hasher atomic.Pointer[review.Hasher]
	snap   atomic.Pointer[Snapshot]
	gen    atomic.Int64
	// writeMu serialises reviews with rebuilds so a review never hashes a
	// half-rebuilt view.
	writeMu sync.Mutex
	subsMu  sync.Mutex
	subs    map[chan int64]struct{}
	closing chan struct{} // closed on shutdown to end SSE streams
	log     *slog.Logger
}

// New loads the content directory once and returns a ready server.
func New(opt Options) (*Server, error) {
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s := &Server{opt: opt, subs: map[chan int64]struct{}{}, closing: make(chan struct{}), log: opt.Logger}
	if err := s.Rebuild(); err != nil {
		return nil, err
	}
	return s, nil
}

// Snapshot returns the current snapshot.
func (s *Server) Snapshot() *Snapshot { return s.snap.Load() }

// Rebuild reloads the content directory and notifies live reload clients.
func (s *Server) Rebuild() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.rebuildLocked()
}

func (s *Server) rebuildLocked() error {
	gen := s.gen.Add(1)
	snap, err := NewSnapshot(s.opt.Dir, s.hasher.Load(), gen)
	if err != nil {
		return err
	}
	if h := s.hasher.Load(); h == nil || h.Root() != snap.Index.RepoRoot {
		s.hasher.Store(review.NewHasher(snap.Index.RepoRoot))
	}
	s.snap.Store(snap)
	s.broadcast(gen)
	s.log.Info("index rebuilt", "generation", gen, "entities", len(snap.Statuses), "errors", snap.Index.ErrorCount())
	return nil
}

// Handler returns the HTTP handler with every route.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/project", s.handleProject)
	mux.HandleFunc("GET /api/entities", s.handleEntities)
	mux.HandleFunc("GET /api/entities/{kind}/{id}", s.handleEntity)
	mux.HandleFunc("POST /api/entities/{kind}/{id}/review", s.handleReview)
	mux.HandleFunc("GET /api/search-index", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Snapshot().SearchIndex()) })
	mux.HandleFunc("GET /api/graph", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Snapshot().Graph()) })
	mux.HandleFunc("GET /api/glossary", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Snapshot().Glossary()) })
	mux.HandleFunc("GET /api/validation", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, s.Snapshot().Validation()) })
	mux.HandleFunc("GET /api/events", s.handleEvents)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { writeError(w, 404, "not found") })
	mux.HandleFunc("GET /content/{path...}", s.handleContent)
	if s.opt.UI != nil {
		mux.Handle("/", s.opt.UI)
	}
	return s.logRequests(s.checkHost(mux))
}

// checkHost refuses requests whose Host header is not a local name when the
// server is bound to loopback. A web page that rebinds its own domain to
// 127.0.0.1 then cannot read the API or post reviews.
func (s *Server) checkHost(next http.Handler) http.Handler {
	if !s.opt.Local {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		host = strings.Trim(strings.ToLower(host), "[]")
		switch host {
		case "localhost", "127.0.0.1", "::1", strings.Trim(strings.ToLower(s.opt.Host), "[]"):
			next.ServeHTTP(w, r)
		default:
			writeError(w, 421, "unexpected Host header; open muninn through localhost")
		}
	})
}

// Caps describes what clients of this server may do.
func (s *Server) Caps() Capabilities {
	c := Capabilities{CanReview: !s.opt.ReadOnly, NeedsToken: s.opt.Token != "", ReadOnlyReason: s.opt.ReadOnlyReason}
	if s.opt.Local {
		c.LocalRepo = s.Snapshot().Index.RepoRoot
	}
	return c
}

func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.Snapshot().Project(s.Caps(), s.opt.Version))
}

func (s *Server) handleEntities(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	writeJSON(w, 200, s.Snapshot().Summaries(Filter{Kind: q.Get("type"), State: q.Get("status"), Role: q.Get("role"), Area: q.Get("area")}))
}

func (s *Server) lookup(w http.ResponseWriter, r *http.Request, snap *Snapshot) *content.Entity {
	kind, ok := content.ParseKind(r.PathValue("kind"))
	if !ok {
		writeError(w, 404, "unknown entity kind")
		return nil
	}
	e := snap.Index.Get(kind, r.PathValue("id"))
	if e == nil {
		writeError(w, 404, "entity not found")
	}
	return e
}

func (s *Server) handleEntity(w http.ResponseWriter, r *http.Request) {
	snap := s.Snapshot()
	if e := s.lookup(w, r, snap); e != nil {
		writeJSON(w, 200, snap.Detail(e))
	}
}

type reviewRequest struct {
	Status string `json:"status"`
	By     string `json:"by"`
	Note   string `json:"note"`
	// Revision is the entity revision the reviewer was looking at. When it
	// no longer matches the files, the review is refused with 409.
	Revision string `json:"revision"`
}

func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeWrite(w, r) {
		return
	}
	var req reviewRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON body: "+err.Error())
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	// Review against a fresh load, so edits that the watcher has not picked
	// up yet are validated and hashed.
	if err := s.rebuildLocked(); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	snap := s.Snapshot()
	e := s.lookup(w, r, snap)
	if e == nil {
		return
	}
	var errs []content.Issue
	for _, is := range snap.Index.IssuesFor(e.Key()) {
		if is.Severity == content.SevError {
			errs = append(errs, is)
		}
	}
	if len(errs) > 0 {
		writeJSON(w, 422, map[string]any{"error": "fix the validation errors of this entity before reviewing it", "issues": errs})
		return
	}
	if req.Revision != "" && req.Revision != snap.Revisions[e.Key()] {
		writeError(w, 409, "this page changed since you opened it; reload and review the current version")
		return
	}
	rv, err := review.Apply(snap.Index, s.hasher.Load(), e, review.Decision{Status: req.Status, By: req.By, Note: req.Note})
	if errors.Is(err, review.ErrInvalid) {
		writeError(w, 400, err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	s.log.Info("review recorded", "entity", e.Key(), "status", rv.Status, "by", rv.By)
	if err := s.rebuildLocked(); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	snap = s.Snapshot()
	writeJSON(w, 200, snap.Detail(snap.Index.Get(e.Kind, e.ID)))
}

// authorizeWrite applies read-only mode, the bearer token, and a same-origin
// check that stops other web pages from posting to a local server.
func (s *Server) authorizeWrite(w http.ResponseWriter, r *http.Request) bool {
	if s.opt.ReadOnly {
		msg := "server is read-only"
		if s.opt.ReadOnlyReason != "" {
			msg += ": " + s.opt.ReadOnlyReason
		}
		writeError(w, 403, msg)
		return false
	}
	if s.opt.Token != "" {
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(s.opt.Token)) != 1 {
			writeError(w, 401, "missing or wrong token")
			return false
		}
	}
	if origin := r.Header.Get("Origin"); origin != "" && !sameOrigin(origin, r.Host) {
		writeError(w, 403, "cross-origin request refused")
		return false
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		writeError(w, 415, "Content-Type must be application/json")
		return false
	}
	return true
}

func sameOrigin(origin, host string) bool {
	_, rest, ok := strings.Cut(origin, "://")
	return ok && strings.EqualFold(rest, host)
}

// contentTypes are the only files served from the content directory: the
// screenshots. Anything else (HTML, SVG) could run script on this origin.
var contentTypes = map[string]string{
	".webp": "image/webp", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
}

// handleContent serves images from the content directory only. The path is
// cleaned and checked to stay inside the directory, and symlinks leading
// outside it are refused.
func (s *Server) handleContent(w http.ResponseWriter, r *http.Request) {
	dir := s.Snapshot().Index.Dir
	rel := r.PathValue("path")
	ctype, ok := contentTypes[strings.ToLower(filepath.Ext(rel))]
	if !ok || rel == "" || strings.Contains(rel, "\\") || strings.Contains(rel, "\x00") {
		http.NotFound(w, r)
		return
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == ".." || strings.HasPrefix(seg, ".") {
			http.NotFound(w, r)
			return
		}
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil || !strings.HasPrefix(real, realDir+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}
	st, err := os.Stat(real)
	if err != nil || st.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeFile(w, r, real)
}

// handleEvents streams "changed" events with the new generation number.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, 500, "streaming unsupported")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("X-Accel-Buffering", "no")
	ch := make(chan int64, 4)
	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()
	defer func() {
		s.subsMu.Lock()
		delete(s.subs, ch)
		s.subsMu.Unlock()
	}()
	_, _ = io.WriteString(w, "retry: 2000\n\n")
	flusher.Flush()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.closing:
			return
		case gen := <-ch:
			if _, err := io.WriteString(w, "event: changed\ndata: "+itoa(gen)+"\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) broadcast(gen int64) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- gen:
		default: // slow client: it will refetch on the next event
		}
	}
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		if r.URL.Path != "/api/events" {
			s.log.Debug("request", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start))
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func itoa(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// Serve runs the HTTP server on ln until ctx ends, then shuts down within 5s.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	// End SSE streams so Shutdown only waits for in-flight requests such as
	// a review being written.
	close(s.closing)
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		_ = srv.Close()
		return err
	}
	return nil
}
