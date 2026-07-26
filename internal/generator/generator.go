package generator

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/pozeydon-code/generator-microservices-go/internal/spec"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

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

	solution, err := buildSolutionView(cfg)
	if err != nil {
		return nil, err
	}
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
	solutionTemplate := "solution-" + solution.SolutionFormat + ".tmpl"
	if err := g.appendRendered(&files, solution.SolutionFileName, solutionTemplate, solution); err != nil {
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
		if err := g.appendRendered(&files, join("src", service.Name, service.ApplicationProject.Directory, "Common", "ValidationBehavior.cs"), "validation-behavior.tmpl", service); err != nil {
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
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, service.WebApiProject.FileName), "webapi-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Common", "ErrorOrProblemMapper.cs"), "webapi-error-mapper.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.ApplicationTestsProject.Directory, service.ApplicationTestsProject.FileName), "application-tests-project.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, service.WebApiTestsProject.FileName), "webapi-tests-project.tmpl", service); err != nil {
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
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Health", "HealthController.cs"), "health-controller.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "TestWebApiFactory.cs"), "webapi-test-factory.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "TestJwtTokens.cs"), "jwt-test-helper.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "HealthControllerTests.cs"), "webapi-health-tests.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("tests", service.Name, service.WebApiTestsProject.Directory, "AuthenticationTests.cs"), "webapi-auth-tests.tmpl", service); err != nil {
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
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "Program.cs"), "program.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "appsettings.json"), "appsettings.tmpl", service); err != nil {
			return nil, err
		}
		if err := g.appendRendered(&files, join("src", service.Name, service.WebApiProject.Directory, "appsettings.Development.json"), "appsettings-development.tmpl", service); err != nil {
			return nil, err
		}

		for _, entity := range service.Entities {
			data := EntityTemplateData{Service: service, Entity: entity}
			entityFiles := []struct{ path, template string }{
				{join("src", service.Name, service.DomainProject.Directory, "Features", entity.FeatureName, entity.Name+".cs"), "entity.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, entity.Name+"Contracts.cs"), "dto.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "I"+entity.Name+"Repository.cs"), "repository-port.tmpl"},
				{join("src", service.Name, service.InfrastructureProject.Directory, "Persistence", "Features", entity.FeatureName, entity.Name+"Repository.cs"), "repository-implementation.tmpl"},
				{join("src", service.Name, service.WebApiProject.Directory, "Controllers", entity.FeatureName, entity.Name+"Controller.cs"), "controller.tmpl"},
				{join("tests", service.Name, service.ApplicationTestsProject.Directory, "Features", entity.FeatureName, entity.Name+"ApplicationTests.cs"), "application-tests.tmpl"},
				{join("tests", service.Name, service.WebApiTestsProject.Directory, "Features", entity.FeatureName, entity.Name+"ControllerTests.cs"), "webapi-tests.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "List", "List"+entity.Name+"Query.cs"), "list-query.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "List", "List"+entity.Name+"QueryHandler.cs"), "list-query-handler.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "List", "List"+entity.Name+"QueryValidator.cs"), "list-query-validator.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "GetById", "Get"+entity.Name+"ByIdQuery.cs"), "get-by-id-query.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "GetById", "Get"+entity.Name+"ByIdQueryHandler.cs"), "get-by-id-query-handler.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Create", "Create"+entity.Name+"Command.cs"), "create-command.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Create", "Create"+entity.Name+"CommandHandler.cs"), "create-command-handler.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Create", "Create"+entity.Name+"CommandValidator.cs"), "create-command-validator.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Update", "Update"+entity.Name+"Command.cs"), "update-command.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Update", "Update"+entity.Name+"CommandHandler.cs"), "update-command-handler.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Update", "Update"+entity.Name+"CommandValidator.cs"), "update-command-validator.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Delete", "Delete"+entity.Name+"Command.cs"), "delete-command.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Delete", "Delete"+entity.Name+"CommandHandler.cs"), "delete-command-handler.tmpl"},
				{join("src", service.Name, service.ApplicationProject.Directory, "Features", entity.FeatureName, "Delete", "Delete"+entity.Name+"CommandValidator.cs"), "delete-command-validator.tmpl"},
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

func join(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}
