package application

import (
	"errors"

	"github.com/pozeydon-code/generator-microservices-go/internal/configloader"
	"github.com/pozeydon-code/generator-microservices-go/internal/generator"
	"github.com/pozeydon-code/generator-microservices-go/internal/output"
	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

type ConfigLoader interface {
	LoadConfig(path string) (spec.Config, error)
}

type ConfigValidator interface {
	ValidateConfig(cfg spec.Config) error
}

type Generator interface {
	Generate(cfg spec.Config) ([]GeneratedFile, error)
}

type OutputWriter interface {
	WriteGeneration(outputDir string, files []GeneratedFile, force bool) (WriteResult, error)
}

type OutputDirectoryResolver interface {
	ResolveOutputDir(outputDir string) (string, error)
}

type Ports struct {
	ConfigLoader            ConfigLoader
	ConfigValidator         ConfigValidator
	Generator               Generator
	OutputWriter            OutputWriter
	OutputDirectoryResolver OutputDirectoryResolver
}

type Service struct {
	ports Ports
}

type GenerateRequest struct {
	ConfigPath string
	OutputDir  string
	Force      bool
}

type GenerationPlan struct {
	OutputDir string
	FileCount int
	Files     []PlannedFile

	generatedFiles []GeneratedFile
}

type PlannedFile struct {
	Path string
}

type GeneratedFile struct {
	Path    string
	Content []byte
}

type WriteResult struct {
	OutputDir string
	Warning   string
}

type GenerateResult struct {
	Plan      GenerationPlan
	OutputDir string
	Warning   string
}

func NewService(ports Ports) Service {
	return Service{ports: ports.withDefaults()}
}

func DefaultService() (Service, error) {
	gen, err := generator.New()
	if err != nil {
		return Service{}, err
	}
	return NewService(Ports{Generator: generatorAdapter{generator: gen}}), nil
}

func (s Service) LoadConfig(path string) (spec.Config, error) {
	return s.ports.ConfigLoader.LoadConfig(path)
}

func (s Service) ValidateConfig(cfg spec.Config) error {
	return s.ports.ConfigValidator.ValidateConfig(cfg)
}

func (s Service) PlanGeneration(request GenerateRequest) (GenerationPlan, error) {
	cfg, err := s.LoadConfig(request.ConfigPath)
	if err != nil {
		return GenerationPlan{}, err
	}
	if err := s.ValidateConfig(cfg); err != nil {
		return GenerationPlan{}, err
	}
	if s.ports.Generator == nil {
		return GenerationPlan{}, errors.New("generator port is required")
	}

	files, err := s.ports.Generator.Generate(cfg)
	if err != nil {
		return GenerationPlan{}, err
	}
	resolvedOutputDir, err := s.ports.OutputDirectoryResolver.ResolveOutputDir(request.OutputDir)
	if err != nil {
		return GenerationPlan{}, err
	}

	plan := GenerationPlan{
		OutputDir:      resolvedOutputDir,
		FileCount:      len(files),
		Files:          make([]PlannedFile, len(files)),
		generatedFiles: files,
	}
	for index, file := range files {
		plan.Files[index] = PlannedFile{Path: file.Path}
	}
	return plan, nil
}

func (s Service) Generate(request GenerateRequest) (GenerateResult, error) {
	plan, err := s.PlanGeneration(request)
	if err != nil {
		return GenerateResult{}, err
	}

	result, err := s.ports.OutputWriter.WriteGeneration(request.OutputDir, plan.generatedFiles, request.Force)
	if err != nil {
		return GenerateResult{}, err
	}
	return GenerateResult{Plan: plan, OutputDir: result.OutputDir, Warning: result.Warning}, nil
}

func (ports Ports) withDefaults() Ports {
	if ports.ConfigLoader == nil {
		ports.ConfigLoader = configLoaderFunc(configloader.LoadJSON)
	}
	if ports.ConfigValidator == nil {
		ports.ConfigValidator = specValidator{}
	}
	if ports.OutputWriter == nil {
		ports.OutputWriter = filesystemWriter{writer: output.NewFilesystemWriter()}
	}
	if ports.OutputDirectoryResolver == nil {
		ports.OutputDirectoryResolver = filesystemOutputResolver{}
	}
	return ports
}

type generatorAdapter struct {
	generator generatedFileProducer
}

func (adapter generatorAdapter) Generate(cfg spec.Config) ([]GeneratedFile, error) {
	files, err := adapter.generator.Generate(cfg)
	if err != nil {
		return nil, err
	}
	generatedFiles := make([]GeneratedFile, len(files))
	for index, file := range files {
		generatedFiles[index] = GeneratedFile{Path: file.Path, Content: file.Content}
	}
	return generatedFiles, nil
}

type generatedFileProducer interface {
	Generate(cfg spec.Config) ([]generator.GeneratedFile, error)
}

type configLoaderFunc func(path string) (spec.Config, error)

func (loader configLoaderFunc) LoadConfig(path string) (spec.Config, error) {
	return loader(path)
}

type specValidator struct{}

func (specValidator) ValidateConfig(cfg spec.Config) error {
	return cfg.Validate()
}

type filesystemWriter struct {
	writer output.FilesystemWriter
}

func (writer filesystemWriter) WriteGeneration(outputDir string, files []GeneratedFile, force bool) (WriteResult, error) {
	generatedFiles := make([]generator.GeneratedFile, len(files))
	for index, file := range files {
		generatedFiles[index] = generator.GeneratedFile{Path: file.Path, Content: file.Content}
	}
	result, err := writer.writer.WriteDetailed(outputDir, generatedFiles, force)
	if err != nil {
		return WriteResult{}, err
	}
	return WriteResult{OutputDir: result.OutputDir, Warning: result.Warning}, nil
}

type filesystemOutputResolver struct{}

func (filesystemOutputResolver) ResolveOutputDir(outputDir string) (string, error) {
	return output.CanonicalPublishPath(outputDir)
}
