package generator

import (
	"bytes"
	"embed"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/spec"
)

//go:embed templates/*/*.tmpl
var templatesFS embed.FS

const (
	rootTemplateDir           = "root"
	domainTemplateDir         = "domain"
	applicationTemplateDir    = "application"
	infrastructureTemplateDir = "infrastructure"
	webAPITemplateDir         = "webapi"
	testsTemplateDir          = "tests"
)

type GeneratedFile struct {
	Path    string
	Content []byte
}

type Generator struct {
	templates *template.Template
}

func New() (*Generator, error) {
	templates, err := template.New("").Option("missingkey=error").ParseFS(templatesFS, "templates/*/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Generator{templates: templates}, nil
}

func (g *Generator) Generate(cfg spec.Config) ([]GeneratedFile, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	solution, err := buildSolutionView(cfg)
	if err != nil {
		return nil, err
	}
	var files []GeneratedFile

	if err := g.appendRendered(&files, "Directory.Build.props", templatePath(rootTemplateDir, "directory-build-props.tmpl"), solution); err != nil {
		return nil, err
	}
	if err := g.appendRendered(&files, "Directory.Packages.props", templatePath(rootTemplateDir, "directory-packages-props.tmpl"), solution); err != nil {
		return nil, err
	}
	if err := g.appendRendered(&files, "README.md", templatePath(rootTemplateDir, "solution-readme.tmpl"), solution); err != nil {
		return nil, err
	}
	if err := g.appendRendered(&files, "microgen.json", templatePath(rootTemplateDir, "solution-metadata.tmpl"), solution); err != nil {
		return nil, err
	}
	solutionTemplate := templatePath(rootTemplateDir, "solution-"+solution.SolutionFormat+".tmpl")
	if err := g.appendRendered(&files, solution.SolutionFileName, solutionTemplate, solution); err != nil {
		return nil, err
	}

	for _, service := range solution.Services {
		if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, service.DomainProject.FileName), templatePath(domainTemplateDir, "domain-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, service.ApplicationProject.FileName), templatePath(applicationTemplateDir, "application-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "ApplicationAssemblyReference.cs"), templatePath(applicationTemplateDir, "application-assembly-reference.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "DependencyInjection.cs"), templatePath(applicationTemplateDir, "application-di.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "Results.cs"), templatePath(applicationTemplateDir, "application-results.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "PaginationPolicy.cs"), templatePath(applicationTemplateDir, "application-pagination-policy.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "Readiness.cs"), templatePath(applicationTemplateDir, "application-readiness.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "ValidationBehavior.cs"), templatePath(applicationTemplateDir, "validation-behavior.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "UnitOfWork.cs"), templatePath(applicationTemplateDir, "application-unit-of-work.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, "Primitives", "DomainResult.cs"), templatePath(domainTemplateDir, "domain-result.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, "Primitives", "DomainReconstitutionException.cs"), templatePath(domainTemplateDir, "domain-reconstitution-exception.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, service.InfrastructureProject.FileName), templatePath(infrastructureTemplateDir, "infrastructure-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, service.WebApiProject.FileName), templatePath(webAPITemplateDir, "webapi-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Common", "Errors", "HttpContextItemKeys.cs"), templatePath(webAPITemplateDir, "http-context-item-keys.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Common", "Errors", "ApiProblemDetailsFactory.cs"), templatePath(webAPITemplateDir, "api-problem-details-factory.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "DependencyInjection.cs"), templatePath(webAPITemplateDir, "webapi-di.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Controllers", "ApiController.cs"), templatePath(webAPITemplateDir, "api-controller.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApplicationTestsProject.Directory, service.ApplicationTestsProject.FileName), templatePath(testsTemplateDir, "application-tests-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, service.WebApiTestsProject.FileName), templatePath(testsTemplateDir, "webapi-tests-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ArchitectureTestsProject.Directory, service.ArchitectureTestsProject.FileName), templatePath(testsTemplateDir, "architecture-tests-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.InfrastructureTestsProject.Directory, service.InfrastructureTestsProject.FileName), templatePath(testsTemplateDir, "infrastructure-tests-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.DomainTestsProject.Directory, service.DomainTestsProject.FileName), templatePath(testsTemplateDir, "domain-tests-project.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.DomainTestsProject.Directory, service.Name+"DomainTests.cs"), templatePath(testsTemplateDir, "domain-tests.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ArchitectureTestsProject.Directory, service.Name+"ArchitectureTests.cs"), templatePath(testsTemplateDir, "architecture-tests.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.InfrastructureTestsProject.Directory, service.Name+"InfrastructureTests.cs"), templatePath(testsTemplateDir, "infrastructure-tests.tmpl"), service); err != nil {
			return nil, err
		}
		for _, valueObject := range service.ValueObjects {
			if err := g.appendRendered(&files, join("src", service.Name, service.DomainProject.Directory, "ValueObjects", valueObject.Name+".cs"), templatePath(domainTemplateDir, "value-object.tmpl"), ValueObjectTemplateData{Service: service, ValueObject: valueObject}); err != nil {
				return nil, err
			}
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Health", "HealthController.cs"), templatePath(webAPITemplateDir, "health-controller.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "TestWebApiFactory.cs"), templatePath(testsTemplateDir, "webapi-test-factory.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "TestJwtTokens.cs"), templatePath(testsTemplateDir, "jwt-test-helper.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "HealthControllerTests.cs"), templatePath(testsTemplateDir, "webapi-health-tests.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "AuthenticationTests.cs"), templatePath(testsTemplateDir, "webapi-auth-tests.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "DependencyInjection.cs"), templatePath(infrastructureTemplateDir, "infrastructure-di.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "Health", "SqlReadinessProbe.cs"), templatePath(infrastructureTemplateDir, "sql-readiness-probe.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", "UnitOfWork.cs"), templatePath(infrastructureTemplateDir, "infrastructure-unit-of-work.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", service.Name+"DbContext.cs"), templatePath(infrastructureTemplateDir, "dbcontext.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", service.Name+"DbContextFactory.cs"), templatePath(infrastructureTemplateDir, "design-time-dbcontext-factory.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", "ValueObjectPreflight.sql"), templatePath(infrastructureTemplateDir, "value-object-preflight-sql.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Program.cs"), templatePath(webAPITemplateDir, "program.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "appsettings.json"), templatePath(webAPITemplateDir, "appsettings.tmpl"), service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "appsettings.Development.json"), templatePath(webAPITemplateDir, "appsettings-development.tmpl"), service); err != nil {
			return nil, err
		}

		for _, entity := range service.Entities {
			data := EntityTemplateData{Service: service, Entity: entity}
			applicationFeaturePath := join("src", service.Name, service.ApplicationProject.Directory, entity.FeatureName)
			entityFiles := []struct{ path, template string }{
				{join("src", service.Name, service.DomainProject.Directory, "Entities", entity.Name+".cs"), templatePath(domainTemplateDir, "entity.tmpl")},
				{join(applicationFeaturePath, "Dtos", entity.Name+"Dto.cs"), templatePath(applicationTemplateDir, "dto.tmpl")},
				{join(applicationFeaturePath, "Dtos", "Create"+entity.Name+"Request.cs"), templatePath(applicationTemplateDir, "create-request.tmpl")},
				{join(applicationFeaturePath, "Dtos", "Update"+entity.Name+"Request.cs"), templatePath(applicationTemplateDir, "update-request.tmpl")},
				{join(applicationFeaturePath, "Interfaces", "I"+entity.Name+"Repository.cs"), templatePath(applicationTemplateDir, "repository-port.tmpl")},
				{join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", "Configurations", entity.Name+"Configuration.cs"), templatePath(infrastructureTemplateDir, "entity-configuration.tmpl")},
				{join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", "Features", entity.FeatureName, entity.Name+"Repository.cs"), templatePath(infrastructureTemplateDir, "repository-implementation.tmpl")},
				{join("src", service.Name, service.WebApiProject.Directory, "Controllers", entity.FeatureName, entity.Name+"Controller.cs"), templatePath(webAPITemplateDir, "controller.tmpl")},
				{join("tests", service.Name, service.ApplicationTestsProject.Directory, "Features", entity.FeatureName, entity.Name+"ApplicationTests.cs"), templatePath(testsTemplateDir, "application-tests.tmpl")},
				{join("tests", service.Name, service.WebApiTestsProject.Directory, "Features", entity.FeatureName, entity.Name+"ControllerTests.cs"), templatePath(testsTemplateDir, "webapi-tests.tmpl")},
				{join(applicationFeaturePath, "Queries", "List", "List"+entity.Name+"Query.cs"), templatePath(applicationTemplateDir, "list-query.tmpl")},
				{join(applicationFeaturePath, "Queries", "List", "List"+entity.Name+"QueryHandler.cs"), templatePath(applicationTemplateDir, "list-query-handler.tmpl")},
				{join(applicationFeaturePath, "Queries", "List", "List"+entity.Name+"QueryValidator.cs"), templatePath(applicationTemplateDir, "list-query-validator.tmpl")},
				{join(applicationFeaturePath, "Queries", "GetById", "Get"+entity.Name+"ByIdQuery.cs"), templatePath(applicationTemplateDir, "get-by-id-query.tmpl")},
				{join(applicationFeaturePath, "Queries", "GetById", "Get"+entity.Name+"ByIdQueryHandler.cs"), templatePath(applicationTemplateDir, "get-by-id-query-handler.tmpl")},
				{join(applicationFeaturePath, "Commands", "Create", "Create"+entity.Name+"Command.cs"), templatePath(applicationTemplateDir, "create-command.tmpl")},
				{join(applicationFeaturePath, "Commands", "Create", "Create"+entity.Name+"CommandHandler.cs"), templatePath(applicationTemplateDir, "create-command-handler.tmpl")},
				{join(applicationFeaturePath, "Commands", "Create", "Create"+entity.Name+"CommandValidator.cs"), templatePath(applicationTemplateDir, "create-command-validator.tmpl")},
				{join(applicationFeaturePath, "Commands", "Update", "Update"+entity.Name+"Command.cs"), templatePath(applicationTemplateDir, "update-command.tmpl")},
				{join(applicationFeaturePath, "Commands", "Update", "Update"+entity.Name+"CommandHandler.cs"), templatePath(applicationTemplateDir, "update-command-handler.tmpl")},
				{join(applicationFeaturePath, "Commands", "Update", "Update"+entity.Name+"CommandValidator.cs"), templatePath(applicationTemplateDir, "update-command-validator.tmpl")},
				{join(applicationFeaturePath, "Commands", "Delete", "Delete"+entity.Name+"Command.cs"), templatePath(applicationTemplateDir, "delete-command.tmpl")},
				{join(applicationFeaturePath, "Commands", "Delete", "Delete"+entity.Name+"CommandHandler.cs"), templatePath(applicationTemplateDir, "delete-command-handler.tmpl")},
				{join(applicationFeaturePath, "Commands", "Delete", "Delete"+entity.Name+"CommandValidator.cs"), templatePath(applicationTemplateDir, "delete-command-validator.tmpl")},
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
	if err := g.templates.ExecuteTemplate(&buffer, path.Base(name), data); err != nil {
		return nil, fmt.Errorf("render %s: %w", name, err)
	}
	return buffer.Bytes(), nil
}

func templatePath(directory, name string) string {
	return directory + "/" + name
}

func join(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}
