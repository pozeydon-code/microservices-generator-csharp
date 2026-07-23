package application

import (
	"encoding/json"
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

type EntitySettings struct {
	ServiceName string
	Entities    []EntityNameSetting
}

type EntityNameSetting struct {
	OriginalName string
	Name         string
}

type FieldSettings struct {
	ServiceName string
	EntityName  string
	Fields      []FieldSetting
}

type FieldSetting struct {
	OriginalName string
	Name         string
	Type         string
}

type ValueObjectSettings struct {
	ServiceName  string
	ValueObjects []ValueObjectNameSetting
}

type ValueObjectNameSetting struct {
	OriginalName string
	Name         string
	Type         string
	Validations  ValidationRuleSettings
}

type ValidationRuleSettings struct {
	Required       *bool
	MinLength      *int
	MaxLength      *int
	Pattern        *string
	ValidExample   *string
	InvalidExample *string
	Minimum        *string
	Maximum        *string
	NotEmpty       *bool
	NotDefault     *bool
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

type UpdateEntitySettingsResult struct {
	Saved     bool
	Config    ConfigSummary
	Plan      GenerationPlan
	PlanError error
}

type UpdateFieldSettingsResult struct {
	Saved     bool
	Config    ConfigSummary
	Plan      GenerationPlan
	PlanError error
}

type UpdateValueObjectSettingsResult struct {
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
	Services            []ServiceSummary
}

type ServiceSummary struct {
	Name                  string
	EntityNames           []string
	ValueObjectNames      []string
	ValueObjects          []ValueObjectSummary
	ValueObjectReferences []ValueObjectReferenceSummary
	Entities              []EntitySummary
}

type ValueObjectSummary struct {
	Name        string
	Type        string
	Validations ValidationRuleSummary
	RulesLabel  string
}

type ValidationRuleSummary struct {
	Required       *bool
	MinLength      *int
	MaxLength      *int
	Pattern        *string
	ValidExample   *string
	InvalidExample *string
	Minimum        *string
	Maximum        *string
	NotEmpty       *bool
	NotDefault     *bool
}

type ValueObjectReferenceSummary struct {
	ValueObjectName string
	EntityName      string
	FieldName       string
}

type EntitySummary struct {
	Name   string
	Fields []FieldSummary
}

type FieldSummary struct {
	Name string
	Type string
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

func (s Service) UpdateEntitySettings(request GenerateRequest, settings EntitySettings) (UpdateEntitySettingsResult, error) {
	cfg, err := s.LoadConfig(request.ConfigPath)
	if err != nil {
		return UpdateEntitySettingsResult{}, err
	}
	serviceIndex, ok := serviceIndexByName(cfg.Services, settings.ServiceName)
	if !ok {
		return UpdateEntitySettingsResult{}, fmt.Errorf("service %q was not found", settings.ServiceName)
	}
	if len(settings.Entities) == 0 {
		return UpdateEntitySettingsResult{}, errors.New("services must keep at least one entity")
	}
	cfg.Services[serviceIndex].Entities = updatedEntities(cfg.Services[serviceIndex].Entities, settings.Entities)
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = spec.ConfigSchemaVersion
	}
	if err := s.ValidateConfig(cfg); err != nil {
		return UpdateEntitySettingsResult{}, err
	}
	if err := s.SaveConfig(request.ConfigPath, cfg); err != nil {
		return UpdateEntitySettingsResult{}, err
	}
	config := summarizeConfig(cfg)
	plan, err := s.planGenerationFromConfig(request, cfg)
	return UpdateEntitySettingsResult{Saved: true, Config: config, Plan: plan, PlanError: err}, nil
}

func (s Service) UpdateFieldSettings(request GenerateRequest, settings FieldSettings) (UpdateFieldSettingsResult, error) {
	cfg, err := s.LoadConfig(request.ConfigPath)
	if err != nil {
		return UpdateFieldSettingsResult{}, err
	}
	serviceIndex, ok := serviceIndexByName(cfg.Services, settings.ServiceName)
	if !ok {
		return UpdateFieldSettingsResult{}, fmt.Errorf("service %q was not found", settings.ServiceName)
	}
	entityIndex, ok := entityIndexByName(cfg.Services[serviceIndex].Entities, settings.EntityName)
	if !ok {
		return UpdateFieldSettingsResult{}, fmt.Errorf("entity %q was not found in service %q", settings.EntityName, settings.ServiceName)
	}
	if len(settings.Fields) == 0 {
		return UpdateFieldSettingsResult{}, errors.New("entities must keep at least one field")
	}
	cfg.Services[serviceIndex].Entities[entityIndex].Fields = updatedFields(cfg.Services[serviceIndex].Entities[entityIndex].Fields, settings.Fields)
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = spec.ConfigSchemaVersion
	}
	if err := s.ValidateConfig(cfg); err != nil {
		return UpdateFieldSettingsResult{}, err
	}
	if err := s.SaveConfig(request.ConfigPath, cfg); err != nil {
		return UpdateFieldSettingsResult{}, err
	}
	config := summarizeConfig(cfg)
	plan, err := s.planGenerationFromConfig(request, cfg)
	return UpdateFieldSettingsResult{Saved: true, Config: config, Plan: plan, PlanError: err}, nil
}

func (s Service) UpdateValueObjectSettings(request GenerateRequest, settings ValueObjectSettings) (UpdateValueObjectSettingsResult, error) {
	cfg, err := s.LoadConfig(request.ConfigPath)
	if err != nil {
		return UpdateValueObjectSettingsResult{}, err
	}
	serviceIndex, ok := serviceIndexByName(cfg.Services, settings.ServiceName)
	if !ok {
		return UpdateValueObjectSettingsResult{}, fmt.Errorf("service %q was not found", settings.ServiceName)
	}
	service := cfg.Services[serviceIndex]
	if err := validateEditableValueObjectTypes(service.ValueObjects, settings.ValueObjects); err != nil {
		return UpdateValueObjectSettingsResult{}, err
	}
	if err := validateSafeValueObjectReferenceChanges(service, settings.ValueObjects); err != nil {
		return UpdateValueObjectSettingsResult{}, err
	}
	cfg.Services[serviceIndex].ValueObjects = updatedValueObjects(service.ValueObjects, settings.ValueObjects)
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = spec.ConfigSchemaVersion
	}
	if err := s.ValidateConfig(cfg); err != nil {
		return UpdateValueObjectSettingsResult{}, err
	}
	if err := s.SaveConfig(request.ConfigPath, cfg); err != nil {
		return UpdateValueObjectSettingsResult{}, err
	}
	config := summarizeConfig(cfg)
	plan, err := s.planGenerationFromConfig(request, cfg)
	return UpdateValueObjectSettingsResult{Saved: true, Config: config, Plan: plan, PlanError: err}, nil
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

func serviceIndexByName(services []spec.Service, name string) (int, bool) {
	for index, service := range services {
		if service.Name == name {
			return index, true
		}
	}
	return 0, false
}

func updatedEntities(existing []spec.Entity, settings []EntityNameSetting) []spec.Entity {
	entities := make([]spec.Entity, 0, len(settings))
	used := make([]bool, len(existing))
	for _, setting := range settings {
		if setting.OriginalName != "" {
			if entity, ok := entityByName(existing, used, setting.OriginalName); ok {
				entity.Name = setting.Name
				entities = append(entities, entity)
				continue
			}
		}
		if entity, ok := entityByName(existing, used, setting.Name); ok {
			entities = append(entities, entity)
			continue
		}
		entities = append(entities, defaultEntityForName(setting.Name))
	}
	return entities
}

func entityByName(entities []spec.Entity, used []bool, name string) (spec.Entity, bool) {
	for index, entity := range entities {
		if used[index] || entity.Name != name {
			continue
		}
		used[index] = true
		return entity, true
	}
	return spec.Entity{}, false
}

func entityIndexByName(entities []spec.Entity, name string) (int, bool) {
	for index, entity := range entities {
		if entity.Name == name {
			return index, true
		}
	}
	return 0, false
}

func updatedFields(existing []spec.Field, settings []FieldSetting) []spec.Field {
	fields := make([]spec.Field, 0, len(settings))
	used := make([]bool, len(existing))
	for _, setting := range settings {
		if setting.OriginalName != "" {
			if field, ok := fieldByName(existing, used, setting.OriginalName); ok {
				field.Name = setting.Name
				field.Type = setting.Type
				fields = append(fields, field)
				continue
			}
		}
		if field, ok := fieldByName(existing, used, setting.Name); ok {
			field.Type = setting.Type
			fields = append(fields, field)
			continue
		}
		fields = append(fields, spec.Field{Name: setting.Name, Type: setting.Type})
	}
	return fields
}

func fieldByName(fields []spec.Field, used []bool, name string) (spec.Field, bool) {
	for index, field := range fields {
		if used[index] || field.Name != name {
			continue
		}
		used[index] = true
		return field, true
	}
	return spec.Field{}, false
}

func updatedValueObjects(existing []spec.ValueObject, settings []ValueObjectNameSetting) []spec.ValueObject {
	valueObjects := make([]spec.ValueObject, 0, len(settings))
	used := make([]bool, len(existing))
	for _, setting := range settings {
		if setting.OriginalName != "" {
			if valueObject, ok := valueObjectByName(existing, used, setting.OriginalName); ok {
				valueObject = valueObjectWithSettings(valueObject, setting)
				valueObjects = append(valueObjects, valueObject)
				continue
			}
		}
		if valueObject, ok := valueObjectByName(existing, used, setting.Name); ok {
			valueObject = valueObjectWithSettings(valueObject, setting)
			valueObjects = append(valueObjects, valueObject)
			continue
		}
		valueObjects = append(valueObjects, valueObjectWithSettings(defaultValueObjectForName(setting.Name), setting))
	}
	return valueObjects
}

func valueObjectWithSettings(valueObject spec.ValueObject, setting ValueObjectNameSetting) spec.ValueObject {
	valueObject.Name = setting.Name
	typeName := strings.TrimSpace(setting.Type)
	if typeName != "" {
		if !editableValueObjectType(typeName) && typeName == valueObject.Type {
			return valueObject
		}
		valueObject.Type = typeName
		valueObject.Validations = validationRulesFromSettings(setting.Validations)
	}
	return valueObject
}

func validateEditableValueObjectTypes(existing []spec.ValueObject, settings []ValueObjectNameSetting) error {
	for _, setting := range settings {
		typeName := strings.TrimSpace(setting.Type)
		if typeName == "" || editableValueObjectType(typeName) {
			continue
		}
		if valueObject, ok := existingValueObjectForSetting(existing, setting); ok && valueObject.Type == typeName {
			continue
		}
		return fmt.Errorf("value object %q type %q is not editable in the basic rules editor", setting.Name, typeName)
	}
	return nil
}

func existingValueObjectForSetting(existing []spec.ValueObject, setting ValueObjectNameSetting) (spec.ValueObject, bool) {
	for _, valueObject := range existing {
		if setting.OriginalName != "" && valueObject.Name == setting.OriginalName {
			return valueObject, true
		}
		if valueObject.Name == setting.Name {
			return valueObject, true
		}
	}
	return spec.ValueObject{}, false
}

func editableValueObjectType(typeName string) bool {
	switch typeName {
	case "string", "decimal", "int", "Guid", "bool":
		return true
	default:
		return false
	}
}

func validationRulesFromSettings(settings ValidationRuleSettings) spec.ValidationRules {
	rules := spec.ValidationRules{
		Required:       settings.Required,
		MinLength:      settings.MinLength,
		MaxLength:      settings.MaxLength,
		Pattern:        nonEmptyStringPtr(settings.Pattern),
		ValidExample:   settings.ValidExample,
		InvalidExample: settings.InvalidExample,
		NotEmpty:       settings.NotEmpty,
		NotDefault:     settings.NotDefault,
	}
	if settings.Minimum != nil && strings.TrimSpace(*settings.Minimum) != "" {
		minimum := json.Number(strings.TrimSpace(*settings.Minimum))
		rules.Minimum = &minimum
	}
	if settings.Maximum != nil && strings.TrimSpace(*settings.Maximum) != "" {
		maximum := json.Number(strings.TrimSpace(*settings.Maximum))
		rules.Maximum = &maximum
	}
	return rules
}

func nonEmptyStringPtr(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func valueObjectByName(valueObjects []spec.ValueObject, used []bool, name string) (spec.ValueObject, bool) {
	for index, valueObject := range valueObjects {
		if used[index] || valueObject.Name != name {
			continue
		}
		used[index] = true
		return valueObject, true
	}
	return spec.ValueObject{}, false
}

func validateSafeValueObjectReferenceChanges(service spec.Service, settings []ValueObjectNameSetting) error {
	referenced := map[string]bool{}
	for _, valueObject := range service.ValueObjects {
		if serviceFieldsReferenceValueObject(service, valueObject.Name) {
			referenced[valueObject.Name] = true
		}
	}
	for valueObjectName := range referenced {
		status := valueObjectIdentityStatusForSettings(settings, valueObjectName)
		switch status {
		case valueObjectIdentityPreserved:
			continue
		case valueObjectIdentityRenamed:
			return fmt.Errorf("value object %q is referenced by entity fields and cannot be renamed", valueObjectName)
		case valueObjectIdentityReplaced:
			return fmt.Errorf("value object %q is referenced by entity fields and cannot be renamed, deleted, or replaced", valueObjectName)
		default:
			return fmt.Errorf("value object %q is referenced by entity fields and cannot be deleted", valueObjectName)
		}
	}
	return nil
}

type valueObjectIdentityStatus int

const (
	valueObjectIdentityDeleted valueObjectIdentityStatus = iota
	valueObjectIdentityPreserved
	valueObjectIdentityRenamed
	valueObjectIdentityReplaced
)

func valueObjectIdentityStatusForSetting(setting ValueObjectNameSetting, valueObjectName string) valueObjectIdentityStatus {
	if setting.OriginalName == valueObjectName && setting.Name == valueObjectName {
		return valueObjectIdentityPreserved
	}
	if setting.OriginalName == valueObjectName {
		return valueObjectIdentityRenamed
	}
	if setting.Name == valueObjectName {
		return valueObjectIdentityReplaced
	}
	return valueObjectIdentityDeleted
}

func valueObjectIdentityStatusForSettings(settings []ValueObjectNameSetting, valueObjectName string) valueObjectIdentityStatus {
	status := valueObjectIdentityDeleted
	for _, setting := range settings {
		settingStatus := valueObjectIdentityStatusForSetting(setting, valueObjectName)
		if settingStatus == valueObjectIdentityPreserved || settingStatus == valueObjectIdentityRenamed {
			return settingStatus
		}
		if settingStatus == valueObjectIdentityReplaced {
			status = valueObjectIdentityReplaced
		}
	}
	return status
}

func serviceFieldsReferenceValueObject(service spec.Service, valueObjectName string) bool {
	for _, entity := range service.Entities {
		for _, field := range entity.Fields {
			if field.Type == valueObjectName {
				return true
			}
		}
	}
	return false
}

func defaultEntityForName(name string) spec.Entity {
	return spec.Entity{
		Name: name,
		Fields: []spec.Field{
			{Name: "Id", Type: "Guid"},
		},
	}
}

func defaultValueObjectForName(name string) spec.ValueObject {
	required := true
	minLength := 1
	maxLength := 100
	validExample := "Sample"
	return spec.ValueObject{
		Name: name,
		Type: "string",
		Validations: spec.ValidationRules{
			Required:     &required,
			MinLength:    &minLength,
			MaxLength:    &maxLength,
			ValidExample: &validExample,
		},
	}
}

func defaultServiceForName(name string) spec.Service {
	entityName := strings.TrimSuffix(name, "Service")
	if entityName == "" {
		entityName = name
	}
	return spec.Service{
		Name: name,
		Entities: []spec.Entity{
			defaultEntityForName(entityName),
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
		Services:            make([]ServiceSummary, len(cfg.Services)),
	}
	for index, service := range cfg.Services {
		summary.ServiceNames[index] = service.Name
		valueObjectNames := make([]string, len(service.ValueObjects))
		valueObjectSet := map[string]bool{}
		valueObjects := make([]ValueObjectSummary, len(service.ValueObjects))
		for valueObjectIndex, valueObject := range service.ValueObjects {
			valueObjectNames[valueObjectIndex] = valueObject.Name
			valueObjectSet[valueObject.Name] = true
			valueObjects[valueObjectIndex] = summarizeValueObject(valueObject)
		}
		summary.Services[index] = ServiceSummary{Name: service.Name, EntityNames: make([]string, len(service.Entities)), ValueObjectNames: valueObjectNames, ValueObjects: valueObjects, Entities: make([]EntitySummary, len(service.Entities))}
		for entityIndex, entity := range service.Entities {
			summary.Services[index].EntityNames[entityIndex] = entity.Name
			summary.Services[index].Entities[entityIndex] = EntitySummary{Name: entity.Name, Fields: make([]FieldSummary, len(entity.Fields))}
			for fieldIndex, field := range entity.Fields {
				summary.Services[index].Entities[entityIndex].Fields[fieldIndex] = FieldSummary{Name: field.Name, Type: field.Type}
				if valueObjectSet[field.Type] {
					summary.Services[index].ValueObjectReferences = append(summary.Services[index].ValueObjectReferences, ValueObjectReferenceSummary{ValueObjectName: field.Type, EntityName: entity.Name, FieldName: field.Name})
				}
			}
		}
		summary.EntityCount += len(service.Entities)
		summary.ValueObjectCount += len(service.ValueObjects)
	}
	return summary
}

func summarizeValueObject(valueObject spec.ValueObject) ValueObjectSummary {
	rules := summarizeValidationRules(valueObject.Validations)
	return ValueObjectSummary{Name: valueObject.Name, Type: valueObject.Type, Validations: rules, RulesLabel: compactRulesLabel(rules)}
}

func summarizeValidationRules(rules spec.ValidationRules) ValidationRuleSummary {
	summary := ValidationRuleSummary{
		Required:       rules.Required,
		MinLength:      rules.MinLength,
		MaxLength:      rules.MaxLength,
		Pattern:        rules.Pattern,
		ValidExample:   rules.ValidExample,
		InvalidExample: rules.InvalidExample,
		NotEmpty:       rules.NotEmpty,
		NotDefault:     rules.NotDefault,
	}
	if rules.Minimum != nil {
		minimum := rules.Minimum.String()
		summary.Minimum = &minimum
	}
	if rules.Maximum != nil {
		maximum := rules.Maximum.String()
		summary.Maximum = &maximum
	}
	return summary
}

func compactRulesLabel(rules ValidationRuleSummary) string {
	parts := []string{}
	if rules.Required != nil && *rules.Required {
		parts = append(parts, "required")
	}
	if rules.MinLength != nil {
		parts = append(parts, fmt.Sprintf("min=%d", *rules.MinLength))
	}
	if rules.MaxLength != nil {
		parts = append(parts, fmt.Sprintf("max=%d", *rules.MaxLength))
	}
	if rules.Pattern != nil && *rules.Pattern != "" {
		parts = append(parts, "pattern")
	}
	if rules.ValidExample != nil && *rules.ValidExample != "" {
		parts = append(parts, "validExample")
	}
	if rules.InvalidExample != nil && *rules.InvalidExample != "" {
		parts = append(parts, "invalidExample")
	}
	if rules.Minimum != nil && *rules.Minimum != "" {
		parts = append(parts, "minimum="+*rules.Minimum)
	}
	if rules.Maximum != nil && *rules.Maximum != "" {
		parts = append(parts, "maximum="+*rules.Maximum)
	}
	if rules.NotEmpty != nil && *rules.NotEmpty {
		parts = append(parts, "notEmpty")
	}
	if rules.NotDefault != nil && *rules.NotDefault {
		parts = append(parts, "notDefault")
	}
	if len(parts) == 0 {
		return "no rules"
	}
	return strings.Join(parts, ", ")
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
