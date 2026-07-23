package application

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/pozeydon-code/generator-microservices-go/internal/configloader"
	"github.com/pozeydon-code/generator-microservices-go/internal/generator"
	"github.com/pozeydon-code/generator-microservices-go/internal/output"
	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

type ConfigLoader interface {
	LoadConfig(path string) (spec.Config, error)
}

type ConfigSaver interface {
	SaveConfig(path string, cfg spec.Config) error
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

type OutputPlanner interface {
	PlanOutput(outputDir string, files []GeneratedFile, force bool) (OutputPlan, error)
}

type TargetFrameworkSuggester interface {
	SuggestedTargetFrameworks() []string
}

type Ports struct {
	ConfigLoader     ConfigLoader
	ConfigSaver      ConfigSaver
	ConfigValidator  ConfigValidator
	Generator        Generator
	OutputWriter     OutputWriter
	OutputPlanner    OutputPlanner
	TargetFrameworks TargetFrameworkSuggester
}

type Service struct {
	ports Ports
}

type GenerateRequest struct {
	ConfigPath         string
	OutputDir          string
	Force              bool
	ConfigBootstrapped bool
}

type SolutionSettings struct {
	SolutionName        string
	SolutionDescription string
	TargetFramework     string
}

type ServiceSettings struct {
	Services     []ServiceNameSetting
	ServiceNames []string
}

type ServiceNameSetting struct {
	OriginalName string
	Name         string
}

type UpdateSolutionSettingsResult struct {
	Saved     bool
	Config    ConfigSummary
	Plan      GenerationPlan
	PlanError error
}

type UpdateServiceSettingsResult struct {
	Saved     bool
	Config    ConfigSummary
	Plan      GenerationPlan
	PlanError error
}

type GenerationPlan struct {
	Config         ConfigSummary
	OutputDir      string
	OutputAction   string
	ForceRequired  bool
	ForceUsed      bool
	FileCount      int
	Files          []PlannedFile
	ExtraFileCount int
	DeletedFiles   []string

	generatedFiles []GeneratedFile
}

type ConfigSummary struct {
	SolutionName        string
	SolutionDescription string
	TargetFramework     string
	SolutionFormat      string
	ServiceCount        int
	EntityCount         int
	ValueObjectCount    int
	ServiceNames        []string
}

type PlannedFile struct {
	Path   string
	Action string
}

type GeneratedFile struct {
	Path    string
	Content []byte
}

type WriteResult struct {
	OutputDir     string
	OutputAction  string
	ForceRequired bool
	ForceUsed     bool
	Warning       string
}

type OutputPlan struct {
	OutputDir     string
	Action        string
	ForceRequired bool
	ForceUsed     bool
	Files         []OutputPlannedFile
	DeletedFiles  []string
}

type OutputPlannedFile struct {
	Path   string
	Action string
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

func (s Service) SaveConfig(path string, cfg spec.Config) error {
	return s.ports.ConfigSaver.SaveConfig(path, cfg)
}

func (s Service) ValidateConfig(cfg spec.Config) error {
	return s.ports.ConfigValidator.ValidateConfig(cfg)
}

func (s Service) TargetFrameworkSuggestions() []string {
	return s.ports.TargetFrameworks.SuggestedTargetFrameworks()
}

func (s Service) CreateStarterConfig(path string) (ConfigSummary, error) {
	if strings.TrimSpace(path) == "" {
		return ConfigSummary{}, errors.New("config path is required")
	}
	if _, err := os.Stat(path); err == nil {
		return ConfigSummary{}, fmt.Errorf("refusing to create starter config because %s already exists", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ConfigSummary{}, fmt.Errorf("stat config: %w", err)
	}
	cfg := starterConfig()
	if err := s.ValidateConfig(cfg); err != nil {
		return ConfigSummary{}, err
	}
	if err := s.SaveConfig(path, cfg); err != nil {
		return ConfigSummary{}, err
	}
	return summarizeConfig(cfg), nil
}

func (s Service) PlanGeneration(request GenerateRequest) (GenerationPlan, error) {
	cfg, err := s.LoadConfig(request.ConfigPath)
	if err != nil {
		return GenerationPlan{}, err
	}
	return s.planGenerationFromConfig(request, cfg)
}

func (s Service) UpdateSolutionSettings(request GenerateRequest, settings SolutionSettings) (UpdateSolutionSettingsResult, error) {
	cfg, err := s.LoadConfig(request.ConfigPath)
	if err != nil {
		return UpdateSolutionSettingsResult{}, err
	}
	cfg.Solution.Name = settings.SolutionName
	cfg.Solution.Description = settings.SolutionDescription
	targetFramework, ok := spec.NormalizeTargetFramework(settings.TargetFramework)
	if !ok {
		targetFramework = settings.TargetFramework
	}
	cfg.Generation.TargetFramework = targetFramework
	cfg.Generation.SolutionFormat = spec.DefaultSolutionFormat(targetFramework)
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = spec.ConfigSchemaVersion
	}
	if err := s.ValidateConfig(cfg); err != nil {
		return UpdateSolutionSettingsResult{}, err
	}
	if err := s.SaveConfig(request.ConfigPath, cfg); err != nil {
		return UpdateSolutionSettingsResult{}, err
	}
	config := summarizeConfig(cfg)
	plan, err := s.planGenerationFromConfig(request, cfg)
	return UpdateSolutionSettingsResult{Saved: true, Config: config, Plan: plan, PlanError: err}, nil
}

func (s Service) UpdateServiceSettings(request GenerateRequest, settings ServiceSettings) (UpdateServiceSettingsResult, error) {
	cfg, err := s.LoadConfig(request.ConfigPath)
	if err != nil {
		return UpdateServiceSettingsResult{}, err
	}
	cfg.Services = updatedServices(cfg.Services, normalizedServiceSettings(settings))
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = spec.ConfigSchemaVersion
	}
	if err := s.ValidateConfig(cfg); err != nil {
		return UpdateServiceSettingsResult{}, err
	}
	if err := s.SaveConfig(request.ConfigPath, cfg); err != nil {
		return UpdateServiceSettingsResult{}, err
	}
	config := summarizeConfig(cfg)
	plan, err := s.planGenerationFromConfig(request, cfg)
	return UpdateServiceSettingsResult{Saved: true, Config: config, Plan: plan, PlanError: err}, nil
}

func normalizedServiceSettings(settings ServiceSettings) []ServiceNameSetting {
	if len(settings.Services) > 0 {
		return settings.Services
	}
	services := make([]ServiceNameSetting, len(settings.ServiceNames))
	for index, name := range settings.ServiceNames {
		services[index] = ServiceNameSetting{Name: name}
	}
	return services
}

func updatedServices(existing []spec.Service, settings []ServiceNameSetting) []spec.Service {
	services := make([]spec.Service, 0, len(settings))
	used := make([]bool, len(existing))
	for _, setting := range settings {
		if setting.OriginalName != "" {
			if service, ok := serviceByName(existing, used, setting.OriginalName); ok {
				service.Name = setting.Name
				services = append(services, service)
				continue
			}
		}
		if service, ok := serviceByName(existing, used, setting.Name); ok {
			services = append(services, service)
			continue
		}
		services = append(services, defaultServiceForName(setting.Name))
	}
	return services
}

func serviceByName(services []spec.Service, used []bool, name string) (spec.Service, bool) {
	for index, service := range services {
		if used[index] || service.Name != name {
			continue
		}
		used[index] = true
		return service, true
	}
	return spec.Service{}, false
}

func defaultServiceForName(name string) spec.Service {
	entityName := strings.TrimSuffix(name, "Service")
	if entityName == "" {
		entityName = name
	}
	return spec.Service{
		Name: name,
		Entities: []spec.Entity{
			{
				Name: entityName,
				Fields: []spec.Field{
					{Name: "Id", Type: "Guid"},
				},
			},
		},
	}
}

func (s Service) planGenerationFromConfig(request GenerateRequest, cfg spec.Config) (GenerationPlan, error) {
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
	outputPlan, err := s.ports.OutputPlanner.PlanOutput(request.OutputDir, files, request.Force)
	if err != nil {
		return GenerationPlan{}, err
	}

	plan := GenerationPlan{
		Config:         summarizeConfig(cfg),
		OutputDir:      outputPlan.OutputDir,
		OutputAction:   outputPlan.Action,
		ForceRequired:  outputPlan.ForceRequired,
		ForceUsed:      outputPlan.ForceUsed,
		FileCount:      len(files),
		Files:          make([]PlannedFile, len(outputPlan.Files)),
		ExtraFileCount: len(outputPlan.DeletedFiles),
		DeletedFiles:   append([]string(nil), outputPlan.DeletedFiles...),
		generatedFiles: files,
	}
	for index, file := range outputPlan.Files {
		plan.Files[index] = PlannedFile{Path: file.Path, Action: file.Action}
	}
	return plan, nil
}

func starterConfig() spec.Config {
	return spec.Config{
		SchemaVersion: spec.ConfigSchemaVersion,
		Generation: spec.GenerationOptions{
			TargetFramework: spec.DefaultTargetFramework,
			SolutionFormat:  spec.DefaultSolutionFormat(spec.DefaultTargetFramework),
		},
		Solution: spec.Solution{
			Name:        "StarterPlatform",
			Description: "Starter microservice platform.",
		},
		Services: []spec.Service{
			{
				Name: "CatalogService",
				Entities: []spec.Entity{
					{
						Name: "Product",
						Fields: []spec.Field{
							{Name: "Id", Type: "Guid"},
							{Name: "Name", Type: "string"},
						},
					},
				},
			},
		},
	}
}

func summarizeConfig(cfg spec.Config) ConfigSummary {
	summary := ConfigSummary{
		SolutionName:        cfg.Solution.Name,
		SolutionDescription: cfg.Solution.Description,
		TargetFramework:     cfg.TargetFramework(),
		SolutionFormat:      cfg.SolutionFormat(),
		ServiceCount:        len(cfg.Services),
		ServiceNames:        make([]string, len(cfg.Services)),
	}
	for index, service := range cfg.Services {
		summary.ServiceNames[index] = service.Name
		summary.EntityCount += len(service.Entities)
		summary.ValueObjectCount += len(service.ValueObjects)
	}
	return summary
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
	plan.OutputDir = result.OutputDir
	plan.OutputAction = result.OutputAction
	plan.ForceRequired = result.ForceRequired
	plan.ForceUsed = result.ForceUsed
	return GenerateResult{Plan: plan, OutputDir: result.OutputDir, Warning: result.Warning}, nil
}

func (ports Ports) withDefaults() Ports {
	if ports.ConfigLoader == nil {
		ports.ConfigLoader = configLoaderFunc(configloader.LoadJSON)
	}
	if ports.ConfigSaver == nil {
		ports.ConfigSaver = configSaverFunc(configloader.SaveJSON)
	}
	if ports.ConfigValidator == nil {
		ports.ConfigValidator = specValidator{}
	}
	if ports.OutputWriter == nil {
		ports.OutputWriter = filesystemWriter{writer: output.NewFilesystemWriter()}
	}
	if ports.OutputPlanner == nil {
		ports.OutputPlanner = filesystemOutputPlanner{}
	}
	if ports.TargetFrameworks == nil {
		ports.TargetFrameworks = dotnetTargetFrameworkSuggester{}
	}
	return ports
}

type dotnetTargetFrameworkSuggester struct{}

func (dotnetTargetFrameworkSuggester) SuggestedTargetFrameworks() []string {
	output, err := exec.Command("dotnet", "--list-sdks").Output()
	if err != nil {
		return spec.SupportedTargetFrameworks()
	}
	frameworks := targetFrameworksFromSDKList(string(output))
	if len(frameworks) == 0 {
		return spec.SupportedTargetFrameworks()
	}
	return frameworks
}

func targetFrameworksFromSDKList(output string) []string {
	seen := map[int]struct{}{}
	for _, line := range strings.Split(output, "\n") {
		version, _, _ := strings.Cut(strings.TrimSpace(line), " ")
		majorText, _, _ := strings.Cut(version, ".")
		major, err := strconv.Atoi(majorText)
		if err != nil {
			continue
		}
		if framework, ok := spec.NormalizeTargetFramework(strconv.Itoa(major)); ok {
			frameworkMajor, _ := spec.TargetFrameworkMajor(framework)
			seen[frameworkMajor] = struct{}{}
		}
	}
	majors := make([]int, 0, len(seen))
	for major := range seen {
		majors = append(majors, major)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(majors)))
	frameworks := make([]string, 0, len(majors))
	for _, major := range majors {
		framework, _ := spec.NormalizeTargetFramework(strconv.Itoa(major))
		frameworks = append(frameworks, framework)
	}
	return frameworks
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

type configSaverFunc func(path string, cfg spec.Config) error

func (saver configSaverFunc) SaveConfig(path string, cfg spec.Config) error {
	return saver(path, cfg)
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
	return WriteResult{
		OutputDir:     result.OutputDir,
		OutputAction:  string(result.Action),
		ForceRequired: result.ForceRequired,
		ForceUsed:     result.ForceUsed,
		Warning:       result.Warning,
	}, nil
}

type filesystemOutputPlanner struct{}

func (filesystemOutputPlanner) PlanOutput(outputDir string, files []GeneratedFile, force bool) (OutputPlan, error) {
	generatedFiles := make([]generator.GeneratedFile, len(files))
	for index, file := range files {
		generatedFiles[index] = generator.GeneratedFile{Path: file.Path, Content: file.Content}
	}
	plan, err := output.PlanOutput(outputDir, generatedFiles, force)
	if err != nil {
		return OutputPlan{}, err
	}
	plannedFiles := make([]OutputPlannedFile, len(plan.Files))
	for index, file := range plan.Files {
		plannedFiles[index] = OutputPlannedFile{Path: file.Path, Action: string(file.Action)}
	}
	return OutputPlan{
		OutputDir:     plan.OutputDir,
		Action:        string(plan.Action),
		ForceRequired: plan.ForceRequired,
		ForceUsed:     plan.ForceUsed,
		Files:         plannedFiles,
		DeletedFiles:  append([]string(nil), plan.DeletedFiles...),
	}, nil
}
