package generator

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

type SolutionTemplateData struct {
	Solution                   spec.Solution
	TargetFramework            string
	SolutionFormat             string
	SolutionFileName           string
	AspNetCorePackageVersion   string
	AspNetCoreTestingVersion   string
	EntityFrameworkCoreVersion string
	SqlClientVersion           string
	CryptographyXmlVersion     string
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
	solutionFormat := cfg.SolutionFormat()
	dependencyPolicy := dependencyPolicyForTargetFramework(targetFramework)
	view := SolutionTemplateData{
		Solution:                   cfg.Solution,
		TargetFramework:            targetFramework,
		SolutionFormat:             solutionFormat,
		SolutionFileName:           cfg.Solution.Name + "." + solutionFormat,
		AspNetCorePackageVersion:   dependencyPolicy.AspNetCorePackageVersion,
		AspNetCoreTestingVersion:   dependencyPolicy.AspNetCoreTestingPackageVersion,
		EntityFrameworkCoreVersion: dependencyPolicy.EntityFrameworkCorePackageVersion,
		SqlClientVersion:           dependencyPolicy.SqlClientPackageVersion,
		CryptographyXmlVersion:     dependencyPolicy.CryptographyXmlPackageVersion,
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

func deterministicGUID(value string) string {
	sum := md5.Sum([]byte(value))
	hexValue := strings.ToUpper(hex.EncodeToString(sum[:]))
	return fmt.Sprintf("{%s-%s-%s-%s-%s}", hexValue[0:8], hexValue[8:12], hexValue[12:16], hexValue[16:20], hexValue[20:32])
}
