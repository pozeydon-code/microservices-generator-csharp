package generator

import "strings"

type ScaffoldPlan struct {
	Commands       []ScaffoldCommand
	PackageEntries []ScaffoldPackageEntry
}

type ScaffoldCommand struct {
	Command string
}

type ScaffoldPackageEntry struct {
	Project string
	Package string
}

func buildScaffoldPlan(solution SolutionTemplateData) ScaffoldPlan {
	plan := ScaffoldPlan{}
	if solution.Gateway.Enabled {
		solutionCommand := []string{"dotnet", "new", "sln"}
		if solution.SolutionFormat == "slnx" {
			solutionCommand = append(solutionCommand, "--format", solution.SolutionFormat)
		}
		solutionCommand = append(solutionCommand, "--name", solution.Solution.Name, "--output", ".")
		plan.Commands = append(plan.Commands,
			ScaffoldCommand{Command: shellCommand(solutionCommand...)},
			newProjectCommand("web", solution.TargetFramework, solution.Gateway.Project),
		)
		for _, project := range solution.Projects {
			plan.Commands = append(plan.Commands, ScaffoldCommand{Command: shellCommand("dotnet", "sln", "./"+solution.Solution.Name+"."+solution.SolutionFormat, "add", "./"+project.Path)})
		}
		plan.PackageEntries = append(plan.PackageEntries, packageEntry(solution.Gateway.Project, "Yarp.ReverseProxy"))
	}

	for _, service := range solution.Services {
		solutionCommand := []string{"dotnet", "new", "sln"}
		if solution.SolutionFormat == "slnx" {
			solutionCommand = append(solutionCommand, "--format", solution.SolutionFormat)
		}
		solutionCommand = append(solutionCommand, "--name", service.Name, "--output", "./"+service.Name)

		plan.Commands = append(plan.Commands,
			ScaffoldCommand{Command: shellCommand(solutionCommand...)},
			newProjectCommand("classlib", solution.TargetFramework, service.DomainProject),
			newProjectCommand("classlib", solution.TargetFramework, service.ApplicationProject),
			newProjectCommand("classlib", solution.TargetFramework, service.InfrastructureProject),
			newProjectCommand("webapi --use-controllers", solution.TargetFramework, service.WebApiProject),
			newProjectCommand("xunit", solution.TargetFramework, service.DomainTestsProject),
			newProjectCommand("xunit", solution.TargetFramework, service.ApplicationTestsProject),
			newProjectCommand("xunit", solution.TargetFramework, service.WebApiTestsProject),
			newProjectCommand("xunit", solution.TargetFramework, service.ArchitectureTestsProject),
			newProjectCommand("xunit", solution.TargetFramework, service.InfrastructureTestsProject),
		)

		for _, project := range []ProjectView{service.DomainProject, service.ApplicationProject, service.InfrastructureProject, service.WebApiProject, service.DomainTestsProject, service.ApplicationTestsProject, service.WebApiTestsProject, service.ArchitectureTestsProject, service.InfrastructureTestsProject} {
			plan.Commands = append(plan.Commands, ScaffoldCommand{Command: shellCommand("dotnet", "sln", "./"+service.Name+"/"+service.SolutionFileName, "add", "./"+project.Path)})
		}

		plan.Commands = append(plan.Commands,
			projectReferenceCommand(service.ApplicationProject, service.DomainProject),
			projectReferenceCommand(service.InfrastructureProject, service.ApplicationProject),
			projectReferenceCommand(service.InfrastructureProject, service.DomainProject),
			projectReferenceCommand(service.WebApiProject, service.ApplicationProject),
			projectReferenceCommand(service.WebApiProject, service.InfrastructureProject),
			projectReferenceCommand(service.DomainTestsProject, service.DomainProject),
			projectReferenceCommand(service.ApplicationTestsProject, service.ApplicationProject),
			projectReferenceCommand(service.ApplicationTestsProject, service.DomainProject),
			projectReferenceCommand(service.WebApiTestsProject, service.WebApiProject),
			projectReferenceCommand(service.WebApiTestsProject, service.ApplicationProject),
			projectReferenceCommand(service.WebApiTestsProject, service.DomainProject),
			projectReferenceCommand(service.ArchitectureTestsProject, service.DomainProject),
			projectReferenceCommand(service.ArchitectureTestsProject, service.ApplicationProject),
			projectReferenceCommand(service.ArchitectureTestsProject, service.WebApiProject),
			projectReferenceCommand(service.ArchitectureTestsProject, service.InfrastructureProject),
			projectReferenceCommand(service.InfrastructureTestsProject, service.InfrastructureProject),
			projectReferenceCommand(service.InfrastructureTestsProject, service.ApplicationProject),
			projectReferenceCommand(service.InfrastructureTestsProject, service.DomainProject),
		)

		plan.PackageEntries = append(plan.PackageEntries,
			packageEntry(service.ApplicationProject, "ErrorOr"),
			packageEntry(service.ApplicationProject, "FluentValidation"),
			packageEntry(service.ApplicationProject, "FluentValidation.DependencyInjectionExtensions"),
			packageEntry(service.ApplicationProject, "MediatR"),
			packageEntry(service.WebApiProject, "Microsoft.AspNetCore.Authentication.JwtBearer"),
			packageEntry(service.WebApiProject, "ErrorOr"),
			packageEntry(service.WebApiProject, "MediatR"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Exporter.OpenTelemetryProtocol"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Extensions.Hosting"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Instrumentation.AspNetCore"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Instrumentation.Http"),
			packageEntry(service.InfrastructureProject, "Microsoft.EntityFrameworkCore.Design"),
			packageEntry(service.InfrastructureProject, "Microsoft.EntityFrameworkCore.Tools"),
			packageEntry(service.InfrastructureProject, "Microsoft.EntityFrameworkCore.SqlServer"),
			packageEntry(service.InfrastructureProject, "Microsoft.Data.SqlClient"),
			packageEntry(service.WebApiTestsProject, "Microsoft.AspNetCore.Mvc.Testing"),
			packageEntry(service.WebApiTestsProject, "System.IdentityModel.Tokens.Jwt"),
		)
		if solution.SupportsOpenApiEndpoints {
			plan.PackageEntries = append(plan.PackageEntries,
				packageEntry(service.WebApiProject, "Microsoft.AspNetCore.OpenApi"),
				packageEntry(service.WebApiProject, "Scalar.AspNetCore"),
			)
		}

		for _, project := range []ProjectView{service.DomainTestsProject, service.ApplicationTestsProject, service.WebApiTestsProject, service.ArchitectureTestsProject, service.InfrastructureTestsProject} {
			plan.PackageEntries = append(plan.PackageEntries,
				packageEntry(project, "Microsoft.NET.Test.Sdk"),
				packageEntry(project, "xunit"),
				packageEntry(project, "xunit.runner.visualstudio"),
			)
		}
		plan.PackageEntries = append(plan.PackageEntries,
			packageEntry(service.InfrastructureTestsProject, "Microsoft.EntityFrameworkCore.SqlServer"),
			packageEntry(service.InfrastructureTestsProject, "Microsoft.Data.SqlClient"),
		)
	}

	return plan
}

func newProjectCommand(templateName, targetFramework string, project ProjectView) ScaffoldCommand {
	parts := append([]string{"dotnet", "new"}, strings.Fields(templateName)...)
	parts = append(parts, "--framework", targetFramework, "--name", project.Name, "--output", "./"+projectDirectory(project.Path), "--no-restore")
	return ScaffoldCommand{Command: shellCommand(parts...)}
}

func projectReferenceCommand(from ProjectView, to ProjectView) ScaffoldCommand {
	return ScaffoldCommand{Command: shellCommand("dotnet", "add", "./"+from.Path, "reference", "./"+to.Path)}
}

func packageEntry(project ProjectView, packageName string) ScaffoldPackageEntry {
	return ScaffoldPackageEntry{Project: project.Path, Package: packageName}
}

func projectDirectory(projectPath string) string {
	for index := len(projectPath) - 1; index >= 0; index-- {
		if projectPath[index] == '/' {
			return projectPath[:index]
		}
	}
	return "."
}

func shellCommand(parts ...string) string {
	quoted := make([]string, len(parts))
	for index, part := range parts {
		if shellLiteral(parts, index) {
			quoted[index] = part
			continue
		}
		quoted[index] = shellQuote(part)
	}
	return strings.Join(quoted, " ")
}

func shellLiteral(parts []string, index int) bool {
	value := parts[index]
	if index == 0 || strings.HasPrefix(value, "--") {
		return true
	}
	if index == 1 {
		switch value {
		case "new", "sln", "add":
			return true
		}
	}
	if len(parts) > 1 && parts[1] == "new" && index == 2 {
		switch value {
		case "sln", "classlib", "web", "webapi", "xunit":
			return true
		}
	}
	if len(parts) > 3 && index == 3 {
		switch {
		case parts[1] == "sln" && value == "add":
			return true
		case parts[1] == "add" && value == "reference":
			return true
		}
	}
	return false
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
