package cli

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func validServeLink(href string) bool {
	lower := strings.ToLower(href)
	return (strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//")) ||
		strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

// servePagesFS hides dot files and requires an index for directories.
type servePagesFS struct{ fs.FS }

func (f servePagesFS) Open(name string) (fs.File, error) {
	for _, segment := range strings.Split(name, "/") {
		if strings.HasPrefix(segment, ".") && segment != "." {
			return nil, fs.ErrNotExist
		}
	}
	file, err := f.FS.Open(name)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if info.IsDir() {
		index, err := fs.Stat(f.FS, path.Join(name, "index.html"))
		if err != nil || !index.Mode().IsRegular() {
			_ = file.Close()
			return nil, fs.ErrNotExist
		}
		return servePagesDir{file}, nil
	}
	return file, nil
}

type servePagesDir struct{ fs.File }

// Never list a directory, even if its index disappears during a request.
func (servePagesDir) ReadDir(int) ([]fs.DirEntry, error) { return nil, fs.ErrNotExist }

type servePagesResponseWriter struct{ http.ResponseWriter }

func (w servePagesResponseWriter) WriteHeader(status int) {
	// FileServer clears cache headers on errors.
	w.Header().Set("Cache-Control", "no-cache")
	w.ResponseWriter.WriteHeader(status)
}

func (s *server) pagesHandler() http.Handler {
	files := http.StripPrefix("/pages/", http.FileServerFS(servePagesFS{s.pagesRoot.FS()}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Referrer-Policy", "same-origin")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if _, err := s.identify(r); err != nil {
			if strings.Contains(r.Header.Get("Accept"), "text/html") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				if r.Method != http.MethodHead {
					_, _ = io.WriteString(w, `<!doctype html><p>Sign in to onigirazu serve first: <a href="/">Sign in</a></p>`)
				}
			} else {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}
			return
		}
		for _, segment := range strings.Split(strings.TrimPrefix(r.URL.Path, "/pages"), "/") {
			if strings.HasPrefix(segment, ".") {
				http.NotFound(w, r)
				return
			}
		}
		if r.URL.Path == "/pages" {
			http.Redirect(w, r, "/pages/", http.StatusMovedPermanently)
			return
		}
		files.ServeHTTP(servePagesResponseWriter{w}, r)
	})
}
