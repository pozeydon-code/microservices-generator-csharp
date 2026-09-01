package generator

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
)

type SolutionTemplateData struct {
	Solution                   spec.Solution
	TargetFramework            string
	SolutionFormat             string
	PackageVersions            []PackageVersion
	MediatRVersion             string
	FluentValidationVersion    string
	ErrorOrVersion             string
	AspNetCorePackageVersion   string
	AspNetCoreTestingVersion   string
	EntityFrameworkCoreVersion string
	SqlClientVersion           string
	CryptographyXmlVersion     string
	SupportsOpenApiEndpoints   bool
	ScaffoldPlan               ScaffoldPlan
	Gateway                    GatewayView
	Services                   []ServiceView
	Projects                   []ProjectView
}

type GatewayView struct {
	Enabled bool
	Project ProjectView
	Routes  []GatewayRouteView
}

type GatewayRouteView struct {
	ServiceName        string
	RouteID            string
	ClusterID          string
	Path               string
	DestinationAddress string
	LocalPort          int
}

type PackageVersion struct {
	Name    string
	Version string
	Comment string
}

type ServiceView struct {
	Name                       string
	SolutionFileName           string
	MediatRVersion             string
	FluentValidationVersion    string
	ErrorOrVersion             string
	Entities                   []EntityView
	DomainProject              ProjectView
	ApplicationProject         ProjectView
	InfrastructureProject      ProjectView
	WebApiProject              ProjectView
	ApplicationTestsProject    ProjectView
	WebApiTestsProject         ProjectView
	ArchitectureTestsProject   ProjectView
	InfrastructureTestsProject ProjectView
	DomainTestsProject         ProjectView
	Projects                   []ProjectView
	ExpectedSchemaItems        int
	ValueObjects               []ValueObjectView
	HasValueObjects            bool
	ReadinessLengthEntity      string
	ReadinessLengthField       string
	ReadinessLengthMax         string
	ReadinessLengthMaxPlusOne  string
	ReadinessLengthRequired    bool
	ReadinessSchemaEntity      string
	ReadinessSchemaField       string
	EnableValueObjectPreflight bool
	SupportsOpenApiEndpoints   bool
}

type ProjectView struct {
	Name         string
	Directory    string
	FileName     string
	Path         string
	SolutionPath string
	GUID         string
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
	Name                     string
	PluralName               string
	FeatureName              string
	Route                    string
	Fields                   []FieldView
	NonIDFields              []FieldView
	RelationshipScalarFields []RelationshipScalarFieldView
	ReferenceNavigations     []ReferenceNavigationView
	CollectionNavigations    []CollectionNavigationView
	EFRelationships          []EFRelationshipView
	ValueObjectFields        []FieldView
	HasValueObjectFields     bool
}

type RelationshipScalarFieldView struct {
	Name              string
	CamelName         string
	Type              string
	ContractType      string
	ValueAccess       string
	Required          bool
	SampleValue       string
	UpdatedValue      string
	DomainSampleValue string
	Assertion         string
}

type ReferenceNavigationView struct {
	Name         string
	TargetEntity string
	Nullable     bool
	Initializer  string
}

type CollectionNavigationView struct {
	Name         string
	TargetEntity string
}

type EFRelationshipView struct {
	PrincipalEntity     string
	DependentEntity     string
	ForeignKeyName      string
	PrincipalNavigation string
	DependentNavigation string
	Required            bool
	IsRequiredCall      string
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

func buildSolutionView(cfg spec.Config) (SolutionTemplateData, error) {
	services := sortedServices(cfg.Services)
	targetFramework := cfg.TargetFramework()
	solutionFormat := cfg.SolutionFormat()
	dependencyPolicy, ok := dependencyPolicyForTargetFramework(targetFramework)
	if !ok {
		return SolutionTemplateData{}, ValidateTargetFrameworkPolicy(targetFramework)
	}
	supportsOpenApiEndpoints := supportsOpenApiEndpoints(targetFramework)
	view := SolutionTemplateData{
		Solution:                   cfg.Solution,
		TargetFramework:            targetFramework,
		SolutionFormat:             solutionFormat,
		PackageVersions:            dependencyPackageVersions(dependencyPolicy, cfg.Generation.Gateway.Enabled),
		MediatRVersion:             dependencyPackageVersion(dependencyPolicy, "MediatR"),
		FluentValidationVersion:    dependencyPackageVersion(dependencyPolicy, "FluentValidation"),
		ErrorOrVersion:             dependencyPackageVersion(dependencyPolicy, "ErrorOr"),
		AspNetCorePackageVersion:   dependencyPackageVersion(dependencyPolicy, "Microsoft.AspNetCore.Authentication.JwtBearer"),
		AspNetCoreTestingVersion:   dependencyPackageVersion(dependencyPolicy, "Microsoft.AspNetCore.Mvc.Testing"),
		EntityFrameworkCoreVersion: dependencyPackageVersion(dependencyPolicy, "Microsoft.EntityFrameworkCore.Design"),
		SqlClientVersion:           dependencyPackageVersion(dependencyPolicy, "Microsoft.Data.SqlClient"),
		CryptographyXmlVersion:     dependencyPackageVersion(dependencyPolicy, "System.Security.Cryptography.Xml"),
		SupportsOpenApiEndpoints:   supportsOpenApiEndpoints,
		Services:                   make([]ServiceView, 0, len(services)),
	}
	for _, service := range services {
		serviceView := ServiceView{
			Name:                       service.Name,
			SolutionFileName:           service.Name + "." + solutionFormat,
			MediatRVersion:             dependencyPackageVersion(dependencyPolicy, "MediatR"),
			FluentValidationVersion:    dependencyPackageVersion(dependencyPolicy, "FluentValidation"),
			ErrorOrVersion:             dependencyPackageVersion(dependencyPolicy, "ErrorOr"),
			EnableValueObjectPreflight: cfg.Generation.EnableValueObjectPreflight,
			SupportsOpenApiEndpoints:   supportsOpenApiEndpoints,
		}
		serviceView.DomainProject = projectView(service.Name, service.Name+".Domain")
		serviceView.ApplicationProject = projectView(service.Name, service.Name+".Application")
		serviceView.InfrastructureProject = projectView(service.Name, service.Name+".Infrastructure")
		serviceView.WebApiProject = projectView(service.Name, service.Name+".WebApi")
		serviceView.ApplicationTestsProject = testProjectView(service.Name, service.Name+".Application.Tests")
		serviceView.DomainTestsProject = testProjectView(service.Name, service.Name+".Domain.Tests")
		serviceView.WebApiTestsProject = testProjectView(service.Name, service.Name+".WebApi.Tests")
		serviceView.ArchitectureTestsProject = testProjectView(service.Name, service.Name+".Architecture.Tests")
		serviceView.InfrastructureTestsProject = testProjectView(service.Name, service.Name+".Infrastructure.Tests")
		serviceView.Projects = []ProjectView{serviceView.DomainProject, serviceView.ApplicationProject, serviceView.InfrastructureProject, serviceView.WebApiProject, serviceView.DomainTestsProject, serviceView.ApplicationTestsProject, serviceView.WebApiTestsProject, serviceView.ArchitectureTestsProject, serviceView.InfrastructureTestsProject}
		sort.Slice(serviceView.Projects, func(i, j int) bool {
			return serviceView.Projects[i].SolutionPath < serviceView.Projects[j].SolutionPath
		})
		serviceView.ValueObjects = valueObjectViews(service.ValueObjects)
		serviceView.HasValueObjects = len(serviceView.ValueObjects) > 0
		valueObjectsByName := map[string]ValueObjectView{}
		for _, valueObject := range serviceView.ValueObjects {
			valueObjectsByName[valueObject.Name] = valueObject
		}
		for _, entity := range sortedEntities(service.Entities) {
			entityView := entityViewWithSortedFields(entity, valueObjectsByName)
			if len(serviceView.Entities) == 0 {
				serviceView.ReadinessSchemaEntity = entityView.Name
				if len(entityView.NonIDFields) > 0 {
					serviceView.ReadinessSchemaField = entityView.NonIDFields[0].Name
				}
			}
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
		applyRelationshipViews(&serviceView, service.CanonicalRelationships())
		view.Services = append(view.Services, serviceView)
		for _, project := range serviceView.Projects {
			view.Projects = append(view.Projects, rootSolutionProjectView(project))
		}
	}
	if cfg.Generation.Gateway.Enabled {
		view.Gateway = gatewayView(cfg.Solution.Name, services)
		view.Projects = append(view.Projects, view.Gateway.Project)
	}
	sort.Slice(view.Projects, func(i, j int) bool { return view.Projects[i].Path < view.Projects[j].Path })
	view.ScaffoldPlan = buildScaffoldPlan(view)
	return view, nil
}

func applyRelationshipViews(serviceView *ServiceView, relationships []spec.CanonicalRelationship) {
	if len(relationships) == 0 {
		return
	}
	sort.Slice(relationships, func(i, j int) bool {
		left := relationships[i]
		right := relationships[j]
		return left.PrincipalEntity+"/"+left.DependentEntity+"/"+left.ForeignKeyName < right.PrincipalEntity+"/"+right.DependentEntity+"/"+right.ForeignKeyName
	})
	entityIndexes := make(map[string]int, len(serviceView.Entities))
	for index, entity := range serviceView.Entities {
		entityIndexes[entity.Name] = index
	}
	for _, relationship := range relationships {
		dependentIndex, hasDependent := entityIndexes[relationship.DependentEntity]
		principalIndex, hasPrincipal := entityIndexes[relationship.PrincipalEntity]
		if !hasDependent || !hasPrincipal {
			continue
		}
		fieldType := relationship.ForeignKeyType
		if relationship.Nullable() {
			fieldType += "?"
		}
		sampleType := strings.TrimSuffix(fieldType, "?")
		sampleValue := sampleValueFor(sampleType, relationship.ForeignKeyName)
		removeExistingRelationshipScalarField(&serviceView.Entities[dependentIndex], relationship.ForeignKeyName)
		serviceView.Entities[dependentIndex].RelationshipScalarFields = append(serviceView.Entities[dependentIndex].RelationshipScalarFields, RelationshipScalarFieldView{
			Name:              relationship.ForeignKeyName,
			CamelName:         camelName(relationship.ForeignKeyName),
			Type:              fieldType,
			ContractType:      fieldType,
			ValueAccess:       relationship.ForeignKeyName,
			Required:          relationship.Required,
			SampleValue:       sampleValue,
			UpdatedValue:      updatedValueFor(sampleType, relationship.ForeignKeyName),
			DomainSampleValue: sampleValue,
			Assertion:         assertionFor(sampleType, sampleValue, relationship.ForeignKeyName),
		})
		initializer := " = null!;"
		if relationship.Nullable() {
			initializer = ""
		}
		serviceView.Entities[dependentIndex].ReferenceNavigations = append(serviceView.Entities[dependentIndex].ReferenceNavigations, ReferenceNavigationView{
			Name:         relationship.DependentNavigation,
			TargetEntity: relationship.PrincipalEntity,
			Nullable:     relationship.Nullable(),
			Initializer:  initializer,
		})
		serviceView.Entities[dependentIndex].EFRelationships = append(serviceView.Entities[dependentIndex].EFRelationships, EFRelationshipView{
			PrincipalEntity:     relationship.PrincipalEntity,
			DependentEntity:     relationship.DependentEntity,
			ForeignKeyName:      relationship.ForeignKeyName,
			PrincipalNavigation: relationship.PrincipalNavigation,
			DependentNavigation: relationship.DependentNavigation,
			Required:            relationship.Required,
			IsRequiredCall:      relationshipIsRequiredCall(relationship.Required),
		})
		serviceView.Entities[principalIndex].CollectionNavigations = append(serviceView.Entities[principalIndex].CollectionNavigations, CollectionNavigationView{
			Name:         relationship.PrincipalNavigation,
			TargetEntity: relationship.DependentEntity,
		})
	}
}

func removeExistingRelationshipScalarField(entityView *EntityView, fieldName string) {
	entityView.Fields = fieldViewsExcept(entityView.Fields, fieldName)
	entityView.NonIDFields = fieldViewsExcept(entityView.NonIDFields, fieldName)
	entityView.ValueObjectFields = fieldViewsExcept(entityView.ValueObjectFields, fieldName)
	entityView.HasValueObjectFields = len(entityView.ValueObjectFields) > 0
}

func fieldViewsExcept(fields []FieldView, fieldName string) []FieldView {
	filtered := fields[:0]
	for _, field := range fields {
		if field.Name != fieldName {
			filtered = append(filtered, field)
		}
	}
	return filtered
}

func relationshipIsRequiredCall(required bool) string {
	if required {
		return ".IsRequired()"
	}
	return ".IsRequired(false)"
}

func gatewayView(solutionName string, services []spec.Service) GatewayView {
	projectName := solutionName + ".Gateway"
	view := GatewayView{
		Enabled: true,
		Project: gatewayProjectView(projectName),
		Routes:  make([]GatewayRouteView, 0, len(services)),
	}
	for index, service := range services {
		serviceRouteName := serviceRoutePrefix(service.Name)
		localPort := defaultGatewayServicePort(index)
		view.Routes = append(view.Routes, GatewayRouteView{
			ServiceName:        service.Name,
			RouteID:            serviceRouteName + "-route",
			ClusterID:          serviceRouteName + "-cluster",
			Path:               "/" + serviceRouteName + "/{**catch-all}",
			DestinationAddress: fmt.Sprintf("http://localhost:%d/", localPort),
			LocalPort:          localPort,
		})
	}
	return view
}

func serviceRoutePrefix(serviceName string) string {
	var builder strings.Builder
	runes := []rune(serviceName)
	for index, current := range runes {
		if index > 0 && isUpperASCII(current) && (!isUpperASCII(runes[index-1]) || (index+1 < len(runes) && isLowerASCII(runes[index+1]))) {
			builder.WriteByte('-')
		}
		builder.WriteRune(current)
	}
	return strings.ToLower(builder.String())
}

func isUpperASCII(value rune) bool {
	return value >= 'A' && value <= 'Z'
}

func isLowerASCII(value rune) bool {
	return value >= 'a' && value <= 'z'
}

func rootProjectView(projectName string) ProjectView {
	directory := projectName
	fileName := projectName + ".csproj"
	solutionPath := join("src", directory, fileName)
	return ProjectView{Name: projectName, Directory: directory, FileName: fileName, Path: solutionPath, SolutionPath: solutionPath, GUID: deterministicGUID(projectName)}
}

func gatewayProjectView(projectName string) ProjectView {
	fileName := projectName + ".csproj"
	solutionPath := join("Gateway", fileName)
	return ProjectView{Name: projectName, Directory: "Gateway", FileName: fileName, Path: solutionPath, SolutionPath: solutionPath, GUID: deterministicGUID(projectName)}
}

func rootSolutionProjectView(project ProjectView) ProjectView {
	project.SolutionPath = project.Path
	return project
}

func defaultGatewayServicePort(sortedIndex int) int {
	return 5100 + sortedIndex
}

func projectView(serviceName, projectName string) ProjectView {
	directory := projectName
	fileName := projectName + ".csproj"
	solutionPath := join("src", directory, fileName)
	path := join(serviceName, solutionPath)
	return ProjectView{Name: projectName, Directory: directory, FileName: fileName, Path: path, SolutionPath: solutionPath, GUID: deterministicGUID(projectName)}
}

func testProjectView(serviceName, projectName string) ProjectView {
	directory := projectName
	fileName := projectName + ".csproj"
	solutionPath := join("tests", directory, fileName)
	path := join(serviceName, solutionPath)
	return ProjectView{Name: projectName, Directory: directory, FileName: fileName, Path: path, SolutionPath: solutionPath, GUID: deterministicGUID(projectName)}
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
			if valueObject.Type == "string" {
				if invalid := patternInvalidSampleFor(valueObject.Validations); invalid != "" {
					view.PatternInvalidValue = csharpStringLiteral(invalid)
				}
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
