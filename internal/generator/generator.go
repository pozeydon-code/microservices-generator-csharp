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

func join(parts ...string) string {
	return filepath.ToSlash(filepath.Join(parts...))
}
