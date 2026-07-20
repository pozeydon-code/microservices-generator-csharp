package generator

import (
	"bytes"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

var (
	decimalDomainMin = mustRat("-79228162514264337593543950335")
	decimalDomainMax = mustRat("79228162514264337593543950335")
)

type GeneratedFile struct {
	Path    string
	Content []byte
}

type Generator struct {
	templates *template.Template
}

func New() (*Generator, error) {
	templates, err := template.New("").Option("missingkey=error").ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Generator{templates: templates}, nil
}

func (g *Generator) Generate(cfg spec.Config) ([]GeneratedFile, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	solution := buildSolutionView(cfg)
	var files []GeneratedFile

	if err := g.appendRendered(&files, "Directory.Build.props", "directory-build-props.tmpl", solution); err != nil {
		return nil, err
	}
	if err := g.appendRendered(&files, "Directory.Packages.props", "directory-packages-props.tmpl", solution); err != nil {
		return nil, err
	}
	if err := g.appendRendered(&files, "README.md", "solution-readme.tmpl", solution); err != nil {
		return nil, err
	}
	if err := g.appendRendered(&files, "microgen.json", "solution-metadata.tmpl", solution); err != nil {
		return nil, err
	}
	if err := g.appendRendered(&files, solution.Solution.Name+".sln", "solution-sln.tmpl", solution); err != nil {
		return nil, err
	}

	for _, service := range solution.Services {
		if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, service.DomainProject.FileName), "domain-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, service.ApplicationProject.FileName), "application-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "Results.cs"), "application-results.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "PaginationPolicy.cs"), "application-pagination-policy.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "Readiness.cs"), "application-readiness.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, "Shared", "DomainResult.cs"), "domain-result.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, "Shared", "DomainReconstitutionException.cs"), "domain-reconstitution-exception.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, service.InfrastructureProject.FileName), "infrastructure-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApiProject.Directory, service.ApiProject.FileName), "api-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApiProject.Directory, "Common", "ValidationProblemMapper.cs"), "api-validation-problem-mapper.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.HostProject.Directory, service.HostProject.FileName), "host-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApplicationTestsProject.Directory, service.ApplicationTestsProject.FileName), "application-tests-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApiTestsProject.Directory, service.ApiTestsProject.FileName), "api-tests-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ArchitectureTestsProject.Directory, service.ArchitectureTestsProject.FileName), "architecture-tests-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.InfrastructureTestsProject.Directory, service.InfrastructureTestsProject.FileName), "infrastructure-tests-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.DomainTestsProject.Directory, service.DomainTestsProject.FileName), "domain-tests-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.DomainTestsProject.Directory, service.Name+"DomainTests.cs"), "domain-tests.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ArchitectureTestsProject.Directory, service.Name+"ArchitectureTests.cs"), "architecture-tests.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.InfrastructureTestsProject.Directory, service.Name+"InfrastructureTests.cs"), "infrastructure-tests.tmpl", service); err != nil {
			return nil, err
		}
		for _, valueObject := range service.ValueObjects {
			if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, "Shared", "ValueObjects", valueObject.Name+".cs"), "value-object.tmpl", ValueObjectTemplateData{Service: service, ValueObject: valueObject}); err != nil {
				return nil, err
			}
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApiProject.Directory, "Health", "HealthEndpoints.cs"), "health-endpoints.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApiTestsProject.Directory, "TestApiFactory.cs"), "api-test-factory.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApiTestsProject.Directory, "TestJwtTokens.cs"), "jwt-test-helper.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApiTestsProject.Directory, "HealthEndpointsTests.cs"), "health-tests.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApiTestsProject.Directory, "AuthenticationTests.cs"), "auth-tests.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "DependencyInjection.cs"), "infrastructure-di.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", service.Name+"DbContext.cs"), "dbcontext.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", "ValueObjectPreflight.sql"), "value-object-preflight-sql.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.HostProject.Directory, "Program.cs"), "program.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.HostProject.Directory, "appsettings.json"), "appsettings.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.HostProject.Directory, "appsettings.Development.json"), "appsettings-development.tmpl", service); err != nil {
			return nil, err
		}

		for _, entity := range service.Entities {
			data := EntityTemplateData{Service: service, Entity: entity}
			entityFiles := []struct{ path, template string }{
				{join("src", service.Name, service.DomainProject.Directory, "Features", entity.FeatureName, entity.Name+".cs"), "entity.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, entity.Name+"Contracts.cs"), "dto.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "I"+entity.Name+"Repository.cs"), "repository-port.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "I"+entity.Name+"UseCases.cs"), "service-interface.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, entity.Name+"UseCases.cs"), "application-service.tmpl"},
				{join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", "Features", entity.FeatureName, entity.Name+"Repository.cs"), "repository-implementation.tmpl"},
				{join("src", service.Name, service.ApiProject.Directory, "Features", entity.FeatureName, entity.Name+"Endpoints.cs"), "endpoints.tmpl"},
				{join("tests", service.Name, service.ApplicationTestsProject.Directory, "Features", entity.FeatureName, entity.Name+"UseCasesTests.cs"), "application-tests.tmpl"},
				{join("tests", service.Name, service.ApiTestsProject.Directory, "Features", entity.FeatureName, entity.Name+"EndpointsTests.cs"), "api-tests.tmpl"},
			}
			for _, file := range entityFiles {
				if err := g.appendRendered(&files, file.path, file.template, data); err != nil {
					return nil, err
				}
			}
		}
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (g *Generator) appendRendered(files *[]GeneratedFile, path, templateName string, data any) error {
	content, err := g.render(templateName, data)
	if err != nil {
		return err
	}
	*files = append(*files, GeneratedFile{Path: path, Content: content})
	return nil
}

func (g *Generator) render(name string, data any) ([]byte, error) {
	var buffer bytes.Buffer
	if err := g.templates.ExecuteTemplate(&buffer, name, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return buffer.Bytes(), nil
}

type SolutionTemplateData struct {
	Solution                   spec.Solution
	TargetFramework            string
	AspNetCorePackageVersion   string
	EntityFrameworkCoreVersion string
	Services                   []ServiceView
	Projects                   []ProjectView
}

type ServiceView struct {
	Name                       string
	Entities                   []EntityView
	DomainProject              ProjectView
	ApplicationProject         ProjectView
	InfrastructureProject      ProjectView
	ApiProject                 ProjectView
	HostProject                ProjectView
	ApplicationTestsProject    ProjectView
	ApiTestsProject            ProjectView
	ArchitectureTestsProject   ProjectView
	InfrastructureTestsProject ProjectView
	DomainTestsProject         ProjectView
	ExpectedSchemaItems        int
	ValueObjects               []ValueObjectView
	HasValueObjects            bool
	ReadinessLengthEntity      string
	ReadinessLengthField       string
	ReadinessLengthMax         string
	ReadinessLengthMaxPlusOne  string
	ReadinessLengthRequired    bool
}

type ProjectView struct {
	Name      string
	Directory string
	FileName  string
	Path      string
	GUID      string
}

type EntityTemplateData struct {
	Service ServiceView
	Entity  EntityView
}

type ValueObjectTemplateData struct {
	Service     ServiceView
	ValueObject ValueObjectView
}

type ValueObjectView struct {
	Name                string
	Type                string
	ParameterType       string
	SampleValue         string
	UpdatedValue        string
	UnequalValue        string
	HasRequired         bool
	MinLength           string
	MaxLength           string
	Pattern             string
	PatternLiteral      string
	Minimum             string
	Maximum             string
	HasNotEmpty         bool
	HasNotDefault       bool
	ColumnMaxLength     string
	InvalidSamples      []InvalidSampleView
	PatternInvalidValue string
}

type InvalidSampleView struct {
	FieldValue string
	Code       string
	Message    string
	TestName   string
}

type EntityView struct {
	Name                 string
	PluralName           string
	FeatureName          string
	Route                string
	Fields               []FieldView
	NonIDFields          []FieldView
	ValueObjectFields    []FieldView
	HasValueObjectFields bool
}

type FieldView struct {
	Name               string
	CamelName          string
	Type               string
	DomainType         string
	ContractType       string
	IsValueObject      bool
	HasRequired        bool
	ValueAccess        string
	ColumnMaxLength    string
	Initializer        string
	SampleValue        string
	UpdatedValue       string
	DomainSampleValue  string
	DomainUpdatedValue string
	InvalidValue       string
	InvalidCode        string
	InvalidMessage     string
	Minimum            string
	Maximum            string
	SQLSampleLiteral   string
	SQLInvalidLiteral  string
	Assertion          string
}

func buildSolutionView(cfg spec.Config) SolutionTemplateData {
	services := sortedServices(cfg.Services)
	targetFramework := cfg.TargetFramework()
	view := SolutionTemplateData{
		Solution:                   cfg.Solution,
		TargetFramework:            targetFramework,
		AspNetCorePackageVersion:   dotNetPackageVersion(targetFramework),
		EntityFrameworkCoreVersion: dotNetPackageVersion(targetFramework),
		Services:                   make([]ServiceView, 0, len(services)),
	}
	for _, service := range services {
		serviceView := ServiceView{Name: service.Name}
		serviceView.DomainProject = projectView(service.Name, service.Name+".Domain")
		serviceView.ApplicationProject = projectView(service.Name, service.Name+".Application")
		serviceView.InfrastructureProject = projectView(service.Name, service.Name+".Infrastructure")
		serviceView.ApiProject = projectView(service.Name, service.Name+".Api")
		serviceView.HostProject = projectView(service.Name, service.Name+".Host")
		serviceView.ApplicationTestsProject = testProjectView(service.Name, service.Name+".Application.Tests")
		serviceView.DomainTestsProject = testProjectView(service.Name, service.Name+".Domain.Tests")
		serviceView.ApiTestsProject = testProjectView(service.Name, service.Name+".Api.Tests")
		serviceView.ArchitectureTestsProject = testProjectView(service.Name, service.Name+".Architecture.Tests")
		serviceView.InfrastructureTestsProject = testProjectView(service.Name, service.Name+".Infrastructure.Tests")
		serviceView.ValueObjects = valueObjectViews(service.ValueObjects)
		serviceView.HasValueObjects = len(serviceView.ValueObjects) > 0
		valueObjectsByName := map[string]ValueObjectView{}
		for _, valueObject := range serviceView.ValueObjects {
			valueObjectsByName[valueObject.Name] = valueObject
		}
		for _, entity := range sortedEntities(service.Entities) {
			entityView := entityViewWithSortedFields(entity, valueObjectsByName)
			if serviceView.ReadinessLengthField == "" {
				for _, field := range entityView.Fields {
					if field.ColumnMaxLength != "" {
						serviceView.ReadinessLengthEntity = entityView.Name
						serviceView.ReadinessLengthField = field.Name
						serviceView.ReadinessLengthMax = field.ColumnMaxLength
						serviceView.ReadinessLengthRequired = field.HasRequired
						if maxLength, err := strconv.Atoi(field.ColumnMaxLength); err == nil {
							serviceView.ReadinessLengthMaxPlusOne = strconv.Itoa(maxLength + 1)
						}
						break
					}
				}
			}
			serviceView.Entities = append(serviceView.Entities, entityView)
			serviceView.ExpectedSchemaItems += len(entityView.Fields) + 1
		}
		view.Services = append(view.Services, serviceView)
		view.Projects = append(view.Projects, serviceView.DomainProject, serviceView.ApplicationProject, serviceView.InfrastructureProject, serviceView.ApiProject, serviceView.HostProject, serviceView.DomainTestsProject, serviceView.ApplicationTestsProject, serviceView.ApiTestsProject, serviceView.ArchitectureTestsProject, serviceView.InfrastructureTestsProject)
	}
	sort.Slice(view.Projects, func(i, j int) bool { return view.Projects[i].Path < view.Projects[j].Path })
	return view
}

func projectView(serviceName, projectName string) ProjectView {
	directory := projectName
	path := join("src", serviceName, directory, projectName+".csproj")
	return ProjectView{Name: projectName, Directory: directory, FileName: projectName + ".csproj", Path: path, GUID: deterministicGUID(projectName)}
}

func testProjectView(serviceName, projectName string) ProjectView {
	directory := projectName
	path := join("tests", serviceName, directory, projectName+".csproj")
	return ProjectView{Name: projectName, Directory: directory, FileName: projectName + ".csproj", Path: path, GUID: deterministicGUID(projectName)}
}

func sortedServices(services []spec.Service) []spec.Service {
	copyOfServices := append([]spec.Service(nil), services...)
	sort.Slice(copyOfServices, func(i, j int) bool { return copyOfServices[i].Name < copyOfServices[j].Name })
	return copyOfServices
}

func sortedEntities(entities []spec.Entity) []spec.Entity {
	copyOfEntities := append([]spec.Entity(nil), entities...)
	sort.Slice(copyOfEntities, func(i, j int) bool { return copyOfEntities[i].Name < copyOfEntities[j].Name })
	return copyOfEntities
}

func valueObjectViews(valueObjects []spec.ValueObject) []ValueObjectView {
	copyOfValueObjects := append([]spec.ValueObject(nil), valueObjects...)
	sort.Slice(copyOfValueObjects, func(i, j int) bool { return copyOfValueObjects[i].Name < copyOfValueObjects[j].Name })
	views := make([]ValueObjectView, 0, len(copyOfValueObjects))
	for _, valueObject := range copyOfValueObjects {
		sample := sampleValueFor(valueObject.Type, valueObject.Name)
		updated := updatedValueFor(valueObject.Type, valueObject.Name)
		if valueObject.Type == "string" && valueObject.Validations.ValidExample != nil {
			sample = csharpStringLiteral(*valueObject.Validations.ValidExample)
			updated = sample
		}
		view := ValueObjectView{
			Name:          valueObject.Name,
			Type:          valueObject.Type,
			ParameterType: parameterTypeFor(valueObject.Type),
			SampleValue:   sample,
			UpdatedValue:  updated,
			HasRequired:   valueObject.Validations.Required != nil && *valueObject.Validations.Required,
			HasNotEmpty:   valueObject.Validations.NotEmpty != nil && *valueObject.Validations.NotEmpty,
			HasNotDefault: valueObject.Validations.NotDefault != nil && *valueObject.Validations.NotDefault,
		}
		if valueObject.Validations.MinLength != nil {
			view.MinLength = fmt.Sprintf("%d", *valueObject.Validations.MinLength)
		}
		if valueObject.Validations.MaxLength != nil {
			view.MaxLength = fmt.Sprintf("%d", *valueObject.Validations.MaxLength)
			view.ColumnMaxLength = view.MaxLength
		}
		if valueObject.Validations.Pattern != nil {
			view.Pattern = *valueObject.Validations.Pattern
			view.PatternLiteral = csharpStringLiteral(*valueObject.Validations.Pattern)
			if valueObject.Validations.InvalidExample != nil {
				view.PatternInvalidValue = csharpStringLiteral(*valueObject.Validations.InvalidExample)
			}
		}
		if valueObject.Validations.Minimum != nil {
			view.Minimum = numberLiteralFor(valueObject.Type, valueObject.Validations.Minimum.String())
		}
		if valueObject.Validations.Maximum != nil {
			view.Maximum = numberLiteralFor(valueObject.Type, valueObject.Validations.Maximum.String())
		}
		if valueObject.Type == "int" || valueObject.Type == "long" || valueObject.Type == "double" || valueObject.Type == "decimal" {
			if view.Minimum != "" {
				view.SampleValue = view.Minimum
				view.UpdatedValue = view.Minimum
				if view.Maximum != "" && view.Maximum != view.Minimum {
					view.UpdatedValue = view.Maximum
					view.UnequalValue = view.Maximum
				}
			} else if view.Maximum != "" {
				view.SampleValue = view.Maximum
				view.UpdatedValue = view.Maximum
			}
		}
		if valueObject.Type == "string" && valueObject.Validations.ValidExample != nil {
			candidate := *valueObject.Validations.ValidExample + "2"
			if candidate != *valueObject.Validations.ValidExample && stringRulesAcceptForGenerator(candidate, valueObject.Validations) {
				view.UpdatedValue = csharpStringLiteral(candidate)
				view.UnequalValue = view.UpdatedValue
			}
		} else if view.UnequalValue == "" && view.UpdatedValue != "" && view.UpdatedValue != view.SampleValue {
			view.UnequalValue = view.UpdatedValue
		}
		view.InvalidSamples = invalidSamplesFor(view)
		views = append(views, view)
	}
	return views
}

func parameterTypeFor(fieldType string) string {
	if fieldType == "string" {
		return "string?"
	}
	return fieldType
}

func numberLiteralFor(fieldType, value string) string {
	canonical := value
	if strings.ContainsAny(value, ".eE") && fieldType == "double" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			canonical = strconv.FormatFloat(f, 'g', -1, 64)
		}
	}
	switch fieldType {
	case "decimal":
		return canonical + "m"
	case "double":
		return canonical + "d"
	case "long":
		return canonical + "L"
	default:
		return canonical
	}
}

func csharpStringLiteral(value string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			builder.WriteString("\\\"")
		case '\\':
			builder.WriteString("\\\\")
		default:
			if r >= 0x20 && r <= 0x7e {
				builder.WriteRune(r)
			} else if r <= 0xffff {
				builder.WriteString(fmt.Sprintf("\\u%04X", r))
			} else {
				builder.WriteString(fmt.Sprintf("\\U%08X", r))
			}
		}
	}
	builder.WriteByte('"')
	return builder.String()
}

func invalidSamplesFor(valueObject ValueObjectView) []InvalidSampleView {
	var samples []InvalidSampleView
	if valueObject.HasRequired {
		samples = append(samples, InvalidSampleView{FieldValue: "\"\"", Code: valueObject.Name + ".Required", Message: valueObject.Name + " is required.", TestName: "Required"})
	}
	if valueObject.MinLength != "" {
		samples = append(samples, InvalidSampleView{FieldValue: "\"x\"", Code: valueObject.Name + ".MinLength", Message: valueObject.Name + " must be at least " + valueObject.MinLength + " characters.", TestName: "MinLength"})
	}
	if valueObject.MaxLength != "" {
		samples = append(samples, InvalidSampleView{FieldValue: "new string('x', " + valueObject.MaxLength + " + 1)", Code: valueObject.Name + ".MaxLength", Message: valueObject.Name + " must be at most " + valueObject.MaxLength + " characters.", TestName: "MaxLength"})
	}
	if valueObject.Pattern != "" {
		invalid := valueObject.PatternInvalidValue
		if invalid == "" {
			invalid = csharpStringLiteral("***")
		}
		samples = append(samples, InvalidSampleView{FieldValue: invalid, Code: valueObject.Name + ".Pattern", Message: valueObject.Name + " has an invalid format.", TestName: "Pattern"})
	}
	if valueObject.Minimum != "" {
		if invalid := lowerBoundInvalid(valueObject.Type, valueObject.Minimum); invalid != "" {
			samples = append(samples, InvalidSampleView{FieldValue: invalid, Code: valueObject.Name + ".Minimum", Message: valueObject.Name + " must be greater than or equal to " + valueObject.Minimum + ".", TestName: "Minimum"})
		}
	}
	if valueObject.Maximum != "" {
		if invalid := upperBoundInvalid(valueObject.Type, valueObject.Maximum); invalid != "" {
			samples = append(samples, InvalidSampleView{FieldValue: invalid, Code: valueObject.Name + ".Maximum", Message: valueObject.Name + " must be less than or equal to " + valueObject.Maximum + ".", TestName: "Maximum"})
		}
	}
	if valueObject.HasNotEmpty {
		samples = append(samples, InvalidSampleView{FieldValue: "Guid.Empty", Code: valueObject.Name + ".NotEmpty", Message: valueObject.Name + " must not be empty.", TestName: "NotEmpty"})
	}
	if valueObject.HasNotDefault {
		samples = append(samples, InvalidSampleView{FieldValue: "default", Code: valueObject.Name + ".NotDefault", Message: valueObject.Name + " must not be the default value.", TestName: "NotDefault"})
	}
	return samples
}

func lowerBoundInvalid(fieldType, minimum string) string {
	if !hasRepresentableLowerInvalid(fieldType, minimum) {
		return ""
	}
	if fieldType == "double" {
		if invalid := adjacentDoubleInvalid(minimum, -1); invalid != "" {
			return invalid
		}
		return ""
	}
	suffix := ""
	if strings.HasSuffix(minimum, "m") || strings.HasSuffix(minimum, "d") || strings.HasSuffix(minimum, "L") {
		suffix = minimum[len(minimum)-1:]
		minimum = strings.TrimSuffix(minimum, suffix)
	}
	return minimum + suffix + " - 1" + suffix
}

func upperBoundInvalid(fieldType, maximum string) string {
	if !hasRepresentableUpperInvalid(fieldType, maximum) {
		return ""
	}
	if fieldType == "double" {
		if invalid := adjacentDoubleInvalid(maximum, 1); invalid != "" {
			return invalid
		}
		return ""
	}
	suffix := ""
	if strings.HasSuffix(maximum, "m") || strings.HasSuffix(maximum, "d") || strings.HasSuffix(maximum, "L") {
		suffix = maximum[len(maximum)-1:]
		maximum = strings.TrimSuffix(maximum, suffix)
	}
	return maximum + suffix + " + 1" + suffix
}

func hasRepresentableLowerInvalid(fieldType, minimum string) bool {
	value, ok := parseCSharpNumberLiteral(minimum)
	if !ok {
		return true
	}
	switch fieldType {
	case "int":
		return value.Cmp(big.NewRat(math.MinInt32, 1)) > 0
	case "long":
		return value.Cmp(big.NewRat(math.MinInt64, 1)) > 0
	case "decimal":
		return value.Cmp(decimalDomainMin) > 0
	case "double":
		parsed, ok := parseFiniteDoubleLiteral(minimum)
		return ok && parsed > -math.MaxFloat64
	default:
		return true
	}
}

func hasRepresentableUpperInvalid(fieldType, maximum string) bool {
	value, ok := parseCSharpNumberLiteral(maximum)
	if !ok {
		return true
	}
	switch fieldType {
	case "int":
		return value.Cmp(big.NewRat(math.MaxInt32, 1)) < 0
	case "long":
		return value.Cmp(big.NewRat(math.MaxInt64, 1)) < 0
	case "decimal":
		return value.Cmp(decimalDomainMax) < 0
	case "double":
		parsed, ok := parseFiniteDoubleLiteral(maximum)
		return ok && parsed < math.MaxFloat64
	default:
		return true
	}
}

func adjacentDoubleInvalid(value string, direction int) string {
	parsed, ok := parseFiniteDoubleLiteral(value)
	if !ok {
		return ""
	}
	toward := math.Inf(direction)
	adjacent := math.Nextafter(parsed, toward)
	if math.IsInf(adjacent, 0) || math.IsNaN(adjacent) {
		return ""
	}
	return strconv.FormatFloat(adjacent, 'g', 17, 64) + "d"
}

func parseFiniteDoubleLiteral(value string) (float64, bool) {
	trimmed := strings.TrimSuffix(value, "d")
	parsed, err := strconv.ParseFloat(trimmed, 64)
	return parsed, err == nil && !math.IsInf(parsed, 0) && !math.IsNaN(parsed)
}

func stringRulesAcceptForGenerator(value string, rules spec.ValidationRules) bool {
	if rules.Required != nil && *rules.Required && strings.TrimSpace(value) == "" {
		return false
	}
	if rules.MinLength != nil && len(value) < *rules.MinLength {
		return false
	}
	if rules.MaxLength != nil && len(value) > *rules.MaxLength {
		return false
	}
	if rules.Pattern != nil {
		re, err := regexp.Compile(*rules.Pattern)
		if err != nil || !re.MatchString(value) {
			return false
		}
	}
	return true
}

func parseCSharpNumberLiteral(value string) (*big.Rat, bool) {
	trimmed := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(value, "m"), "d"), "L")
	r, ok := new(big.Rat).SetString(trimmed)
	return r, ok
}

func mustRat(value string) *big.Rat {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		panic("invalid rational constant " + value)
	}
	return r
}

func sqlLiteralFor(fieldType, csharpValue string) string {
	switch fieldType {
	case "string":
		return "N'" + strings.ReplaceAll(strings.Trim(csharpValue, "\""), "'", "''") + "'"
	case "Guid":
		return "'" + strings.Split(strings.Split(csharpValue, "\"")[1], "\"")[0] + "'"
	case "DateTime":
		if strings.Contains(csharpValue, "2024, 2, 1") {
			return "'2024-02-01T00:00:00'"
		}
		return "'2024-01-01T00:00:00'"
	case "bool":
		if csharpValue == "true" {
			return "1"
		}
		return "0"
	case "decimal", "double", "long":
		value := strings.TrimRight(strings.TrimRight(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(csharpValue, "m"), "d"), "L"), "0"), ".")
		if value == "" || value == "-" {
			return "0"
		}
		return value
	default:
		return csharpValue
	}
}

func sqlInvalidLiteralFor(fieldType, code, csharpValue string) string {
	switch fieldType {
	case "string":
		if strings.HasSuffix(code, ".MaxLength") {
			return "REPLICATE(N'x', 4096)"
		}
		return sqlLiteralFor(fieldType, csharpValue)
	case "Guid":
		return "'00000000-0000-0000-0000-000000000000'"
	case "DateTime":
		return "'0001-01-01T00:00:00'"
	case "decimal", "double", "long":
		return strings.NewReplacer("m", "", "d", "", "L", "").Replace(csharpValue)
	default:
		return csharpValue
	}
}

func entityViewWithSortedFields(entity spec.Entity, valueObjects map[string]ValueObjectView) EntityView {
	fields := append([]spec.Field(nil), entity.Fields...)
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

	pluralName := pluralize(entity.Name)
	view := EntityView{Name: entity.Name, PluralName: pluralName, FeatureName: pluralName, Route: routeName(pluralName)}
	for _, field := range fields {
		primitiveType := field.Type
		domainType := field.Type
		contractType := field.Type
		isValueObject := false
		valueAccess := field.Name
		if vo, ok := valueObjects[field.Type]; ok {
			primitiveType = vo.Type
			domainType = vo.Name
			contractType = vo.Type
			isValueObject = true
			valueAccess = field.Name + ".Value"
			view.HasValueObjectFields = true
		}
		sampleValue := sampleValueFor(primitiveType, field.Name)
		domainSampleValue := sampleValue
		if isValueObject {
			sampleValue = valueObjects[field.Type].SampleValue
		}
		updatedValue := updatedValueFor(primitiveType, field.Name)
		domainUpdatedValue := updatedValue
		if isValueObject {
			updatedValue = valueObjects[field.Type].UpdatedValue
		}
		if isValueObject {
			domainSampleValue = field.Type + ".Create(" + sampleValue + ").Value!"
			domainUpdatedValue = field.Type + ".Create(" + updatedValue + ").Value!"
		}
		columnMaxLength := ""
		if isValueObject {
			columnMaxLength = valueObjects[field.Type].ColumnMaxLength
		}
		invalidValue := sampleValue
		invalidCode := ""
		invalidMessage := ""
		minimum := ""
		maximum := ""
		if isValueObject && len(valueObjects[field.Type].InvalidSamples) > 0 {
			invalidValue = valueObjects[field.Type].InvalidSamples[0].FieldValue
			invalidCode = valueObjects[field.Type].InvalidSamples[0].Code
			invalidMessage = valueObjects[field.Type].InvalidSamples[0].Message
		}
		if isValueObject {
			minimum = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(valueObjects[field.Type].Minimum, "m"), "d"), "L")
			maximum = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(valueObjects[field.Type].Maximum, "m"), "d"), "L")
		}
		hasRequired := false
		if isValueObject {
			hasRequired = valueObjects[field.Type].HasRequired
		}
		fieldView := FieldView{Name: field.Name, CamelName: camelName(field.Name), Type: field.Type, DomainType: domainType, ContractType: contractType, IsValueObject: isValueObject, HasRequired: hasRequired, ValueAccess: valueAccess, ColumnMaxLength: columnMaxLength, Initializer: initializerFor(domainType), SampleValue: sampleValue, UpdatedValue: updatedValue, DomainSampleValue: domainSampleValue, DomainUpdatedValue: domainUpdatedValue, InvalidValue: invalidValue, InvalidCode: invalidCode, InvalidMessage: invalidMessage, Minimum: minimum, Maximum: maximum, SQLSampleLiteral: sqlLiteralFor(primitiveType, sampleValue), SQLInvalidLiteral: sqlInvalidLiteralFor(primitiveType, invalidCode, invalidValue), Assertion: assertionFor(contractType, sampleValue, field.Name)}
		view.Fields = append(view.Fields, fieldView)
		if field.Name != "Id" {
			view.NonIDFields = append(view.NonIDFields, fieldView)
		}
		if fieldView.IsValueObject {
			view.ValueObjectFields = append(view.ValueObjectFields, fieldView)
		}
	}
	return view
}

func camelName(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToLower(name[:1]) + name[1:]
}

func assertionFor(fieldType, expected, name string) string {
	if fieldType == "bool" {
		if expected == "true" {
			return fmt.Sprintf("Assert.True(created.%s);", name)
		}
		return fmt.Sprintf("Assert.False(created.%s);", name)
	}
	return fmt.Sprintf("Assert.Equal(%s, created.%s);", expected, name)
}

func sampleValueFor(fieldType, name string) string {
	switch fieldType {
	case "string":
		return fmt.Sprintf("\"%s Value\"", name)
	case "bool":
		return "true"
	case "decimal":
		return "12.34m"
	case "double":
		return "12.34d"
	case "int":
		return "12"
	case "long":
		return "12L"
	case "Guid":
		if name == "Id" {
			return "Guid.Parse(\"11111111-1111-1111-1111-111111111111\")"
		}
		return "Guid.Parse(\"00000000-0000-0000-0000-000000000001\")"
	case "DateTime":
		return "new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc)"
	default:
		return "default"
	}
}

func updatedValueFor(fieldType, name string) string {
	switch fieldType {
	case "string":
		return fmt.Sprintf("\"Updated %s\"", name)
	case "bool":
		return "false"
	case "decimal":
		return "56.78m"
	case "double":
		return "56.78d"
	case "int":
		return "34"
	case "long":
		return "34L"
	case "Guid":
		return "Guid.Parse(\"00000000-0000-0000-0000-000000000002\")"
	case "DateTime":
		return "new DateTime(2024, 2, 1, 0, 0, 0, DateTimeKind.Utc)"
	default:
		return "default"
	}
}

func initializerFor(fieldType string) string {
	switch fieldType {
	case "string":
		return " = string.Empty;"
	case "bool", "DateTime", "decimal", "double", "Guid", "int", "long":
		return ""
	default:
		return " = null!;"
	}
}

func deterministicGUID(value string) string {
	sum := md5.Sum([]byte(value))
	hexValue := strings.ToUpper(hex.EncodeToString(sum[:]))
	return fmt.Sprintf("{%s-%s-%s-%s-%s}", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}

func join(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}
