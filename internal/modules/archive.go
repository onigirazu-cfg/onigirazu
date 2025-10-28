package modules

import (
	"archive/tar"
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"compress/gzip"

	"github.com/onigirazu-cfg/onigirazu/internal/executor"
	"github.com/onigirazu-cfg/onigirazu/pkg/types"
)

// ArchiveModule handles file/directory archiving operations
type ArchiveModule struct {
	BaseModule
}

// NewArchiveModule creates a new archive module instance
func NewArchiveModule() *ArchiveModule {
	return &ArchiveModule{
		BaseModule: BaseModule{
			name:        "archive",
			description: "Creates a compressed archive of one or more files or directories",
		},
	}
}

// GetDescription returns the module description
func (m *ArchiveModule) GetDescription() string {
	return m.description
}

// Execute runs the archive module
func (m *ArchiveModule) Execute(ctx context.Context, host types.Host, args map[string]interface{}) (types.TaskResult, error) {
	result := types.TaskResult{
		Host:      host.Name,
		Module:    m.name,
		Timestamp: time.Now(),
		Output:    make(map[string]interface{}),
	}

	startTime := time.Now()
	defer func() {
		result.Duration = time.Since(startTime)
	}()

	// Validate arguments
	if err := m.Validate(args); err != nil {
		result.Failed = true
		result.Error = err.Error()
		return result, err
	}

	// Get arguments
	paths := m.getPaths(args)
	dest := getStringArg(args, "dest", "")
	format := getStringArg(args, "format", "gz")
	remove := getBoolArg(args, "remove", false)
	excludePaths := m.getExcludePaths(args)
	_ = getBoolArg(args, "force_archive", false) // force_archive reserved for future use

	// Initialize executor for remote execution
	exec, err := executor.NewCommandExecutor(host)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to create executor: %v", err)
		return result, err
	}
	defer exec.Close()

	// Collect files from paths (support globs)
	filesToArchive, err := m.collectFiles(paths, excludePaths)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to collect files: %v", err)
		return result, err
	}

	if len(filesToArchive) == 0 {
		result.Failed = true
		result.Error = "no files found matching the specified paths"
		return result, fmt.Errorf("no files found")
	}

	result.Output["files"] = filesToArchive
	result.Output["file_count"] = len(filesToArchive)

	// Create archive
	archiveSize, err := m.createArchive(dest, format, filesToArchive)
	if err != nil {
		result.Failed = true
		result.Error = fmt.Sprintf("failed to create archive: %v", err)
		return result, err
	}

	result.Output["dest"] = dest
	result.Output["format"] = format
	result.Output["size"] = archiveSize
	result.Changed = true

	// Remove source files if requested
	if remove {
		removedFiles := []string{}
		for _, file := range filesToArchive {
			if err := os.RemoveAll(file); err != nil {
				result.Error = fmt.Sprintf("warning: failed to remove %s: %v", file, err)
			} else {
				removedFiles = append(removedFiles, file)
			}
		}
		result.Output["removed_files"] = removedFiles
	}

	result.Success = true
	return result, nil
}

// Validate validates the module arguments
func (m *ArchiveModule) Validate(args map[string]interface{}) error {
	if err := m.BaseModule.Validate(args); err != nil {
		return err
	}

	// Check required arguments
	paths := m.getPaths(args)
	if len(paths) == 0 {
		return fmt.Errorf("'path' is required (can be string or list)")
	}

	if dest, ok := args["dest"]; !ok || dest == "" {
		return fmt.Errorf("'dest' is required")
	}

	// Validate format
	format := getStringArg(args, "format", "gz")
	validFormats := map[string]bool{"tar": true, "gz": true, "bz2": true, "xz": true, "zip": true}
	if !validFormats[format] {
		return fmt.Errorf("invalid format '%s', must be one of: tar, gz, bz2, xz, zip", format)
	}

	return nil
}

// getPaths extracts path argument (can be string or list)
func (m *ArchiveModule) getPaths(args map[string]interface{}) []string {
	var paths []string

	if pathArg, ok := args["path"]; ok {
		switch v := pathArg.(type) {
		case string:
			if v != "" {
				paths = append(paths, v)
			}
		case []interface{}:
			for _, p := range v {
				if str, ok := p.(string); ok && str != "" {
					paths = append(paths, str)
				}
			}
		}
	}

	return paths
}

// getExcludePaths extracts exclude_path argument
func (m *ArchiveModule) getExcludePaths(args map[string]interface{}) []string {
	var excludePaths []string

	if excludeArg, ok := args["exclude_path"]; ok {
		switch v := excludeArg.(type) {
		case string:
			if v != "" {
				excludePaths = append(excludePaths, v)
			}
		case []interface{}:
			for _, p := range v {
				if str, ok := p.(string); ok && str != "" {
					excludePaths = append(excludePaths, str)
				}
			}
		}
	}

	return excludePaths
}

// collectFiles collects all files matching the paths (with glob support)
func (m *ArchiveModule) collectFiles(paths, excludePaths []string) ([]string, error) {
	var allFiles []string
	excludeMap := make(map[string]bool)

	// Build exclude map
	for _, exclude := range excludePaths {
		matches, err := filepath.Glob(exclude)
		if err == nil {
			for _, match := range matches {
				excludeMap[match] = true
			}
		}
	}

	// Collect files from all paths
	for _, path := range paths {
		matches, err := filepath.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern '%s': %v", path, err)
		}

		if len(matches) == 0 {
			// If no glob matches, try as literal path
			if _, err := os.Stat(path); err == nil {
				matches = []string{path}
			}
		}

		for _, match := range matches {
			if !excludeMap[match] {
				allFiles = append(allFiles, match)
			}
		}
	}

	return allFiles, nil
}

// createArchive creates the archive file in the specified format
func (m *ArchiveModule) createArchive(dest, format string, files []string) (int64, error) {
	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(dest)
	if destDir != "." && destDir != "" {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return 0, fmt.Errorf("failed to create destination directory: %v", err)
		}
	}

	// Create archive file
	archiveFile, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("failed to create archive file: %v", err)
	}
	defer archiveFile.Close()

	switch format {
	case "tar":
		return m.createTarArchive(archiveFile, files)
	case "gz":
		return m.createGzArchive(archiveFile, files)
	case "bz2":
		return m.createBz2Archive(archiveFile, files)
	case "zip":
		return m.createZipArchive(dest, files)
	default:
		return 0, fmt.Errorf("unsupported format: %s", format)
	}
}

// createTarArchive creates an uncompressed tar archive
func (m *ArchiveModule) createTarArchive(w io.Writer, files []string) (int64, error) {
	tw := tar.NewWriter(w)
	defer tw.Close()

	for _, file := range files {
		if err := m.addToTar(tw, file, ""); err != nil {
			return 0, err
		}
	}

	if f, ok := w.(*os.File); ok {
		stat, _ := f.Stat()
		return stat.Size(), nil
	}
	return 0, nil
}

// createGzArchive creates a gzip-compressed tar archive
func (m *ArchiveModule) createGzArchive(w io.Writer, files []string) (int64, error) {
	gw := gzip.NewWriter(w)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, file := range files {
		if err := m.addToTar(tw, file, ""); err != nil {
			return 0, err
		}
	}

	tw.Close()
	gw.Close()

	if f, ok := w.(*os.File); ok {
		stat, _ := f.Stat()
		return stat.Size(), nil
	}
	return 0, nil
}

// createBz2Archive creates a bzip2-compressed tar archive
func (m *ArchiveModule) createBz2Archive(w io.Writer, files []string) (int64, error) {
	// Note: bzip2 doesn't have a streaming writer in stdlib
	// For now, we'll create tar and use bzip2 command if available
	// This is a simplified version
	return 0, fmt.Errorf("bz2 format requires external bzip2 tool")
}

// createZipArchive creates a zip archive
func (m *ArchiveModule) createZipArchive(dest string, files []string) (int64, error) {
	zipFile, err := os.Create(dest)
	if err != nil {
		return 0, fmt.Errorf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	for _, file := range files {
		if err := m.addToZip(zw, file, ""); err != nil {
			return 0, err
		}
	}

	zw.Close()

	stat, err := os.Stat(dest)
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

// addToTar adds a file or directory to tar archive
func (m *ArchiveModule) addToTar(tw *tar.Writer, path, prefix string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	var link string
	if info.Mode()&os.ModeSymlink != 0 {
		link, err = os.Readlink(path)
		if err != nil {
			return err
		}
	}

	header, err := tar.FileInfoHeader(info, link)
	if err != nil {
		return err
	}

	header.Name = filepath.Join(prefix, filepath.Base(path))

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := m.addToTar(tw, filepath.Join(path, entry.Name()), header.Name); err != nil {
				return err
			}
		}
		return nil
	}

	if info.Mode()&os.ModeSymlink == 0 {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
	}

	return nil
}

// addToZip adds a file or directory to zip archive
func (m *ArchiveModule) addToZip(zw *zip.Writer, path, prefix string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filepath.Join(prefix, filepath.Base(path))

	if info.IsDir() {
		header.Name += "/"
		_, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := m.addToZip(zw, filepath.Join(path, entry.Name()), header.Name); err != nil {
				return err
			}
		}
		return nil
	}

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	return err
}
