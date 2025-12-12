package folderstructure

import (
	"fmt"
	"time"
)

type Resolver interface {
	Resolve(resourcePath string, projectPath string) *ResolutionResult
	ResolveInDir(dirPath string, projectPath string) ([]*ResolutionResult, error)
	ClearCache()
	ValidatePath(basePath string, resourcePath string) error
}

type FileResolver struct {
	*BaseResolver
}

func NewFileResolver(detector *Detector) *FileResolver {
	config := ResolverConfig{
		DirectoryName: "files",
		CacheKeyPrefix: "file",
		HasDirectoryField: func(ps *ProjectStructure) bool { return ps.HasFiles },
		TTL: 1 * time.Hour,
		MaxCacheSize: 1000,
		NotFoundMessage: func(path string) string {
			return fmt.Sprintf("file not found: %s", path)
		},
	}
	return &FileResolver{
		BaseResolver: NewBaseResolver(detector, config),
	}
}

func (fr *FileResolver) ResolveFile(filePath string, projectPath string) *ResolutionResult {
	return fr.Resolve(filePath, projectPath)
}

func (fr *FileResolver) ResolveFilesInDir(dirPath string, projectPath string) ([]*ResolutionResult, error) {
	return fr.ResolveInDir(dirPath, projectPath)
}

func (fr *FileResolver) ValidateFilePath(basePath string, filePath string) error {
	return fr.ValidatePath(basePath, filePath)
}

type TemplateResolver struct {
	*BaseResolver
}

func NewTemplateResolver(detector *Detector) *TemplateResolver {
	config := ResolverConfig{
		DirectoryName: "templates",
		CacheKeyPrefix: "template",
		HasDirectoryField: func(ps *ProjectStructure) bool { return ps.HasTemplates },
		TTL: 30 * time.Minute,
		MaxCacheSize: 500,
		NotFoundMessage: func(path string) string {
			return fmt.Sprintf("template not found: %s", path)
		},
	}
	return &TemplateResolver{
		BaseResolver: NewBaseResolver(detector, config),
	}
}

func (tr *TemplateResolver) ResolveTemplate(templatePath string, projectPath string) *ResolutionResult {
	return tr.Resolve(templatePath, projectPath)
}

func (tr *TemplateResolver) ResolveTemplatesInDir(dirPath string, projectPath string) ([]*ResolutionResult, error) {
	return tr.ResolveInDir(dirPath, projectPath)
}

func (tr *TemplateResolver) ValidateTemplatePath(basePath string, templatePath string) error {
	return tr.ValidatePath(basePath, templatePath)
}
