package generator

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
	plan := ScaffoldPlan{
		Commands: []ScaffoldCommand{{Command: "dotnet new sln --format " + solution.SolutionFormat + " --name " + solution.Solution.Name}},
	}

	for _, service := range solution.Services {
		plan.Commands = append(plan.Commands,
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
			plan.Commands = append(plan.Commands, ScaffoldCommand{Command: "dotnet sln ./" + solution.SolutionFileName + " add ./" + project.Path})
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
			packageEntry(service.ApplicationProject, "MediatR"),
			packageEntry(service.WebApiProject, "Microsoft.AspNetCore.Authentication.JwtBearer"),
			packageEntry(service.WebApiProject, "ErrorOr"),
			packageEntry(service.WebApiProject, "FluentValidation.DependencyInjectionExtensions"),
			packageEntry(service.WebApiProject, "MediatR"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Exporter.OpenTelemetryProtocol"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Extensions.Hosting"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Instrumentation.AspNetCore"),
			packageEntry(service.WebApiProject, "OpenTelemetry.Instrumentation.Http"),
			packageEntry(service.InfrastructureProject, "Microsoft.EntityFrameworkCore.Design"),
			packageEntry(service.InfrastructureProject, "Microsoft.EntityFrameworkCore.SqlServer"),
			packageEntry(service.InfrastructureProject, "Microsoft.Data.SqlClient"),
			packageEntry(service.WebApiTestsProject, "Microsoft.AspNetCore.Mvc.Testing"),
			packageEntry(service.WebApiTestsProject, "System.IdentityModel.Tokens.Jwt"),
		)

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
	return ScaffoldCommand{Command: "dotnet new " + templateName + " --framework " + targetFramework + " --name " + project.Name + " --output ./" + projectDirectory(project.Path)}
}

func projectReferenceCommand(from ProjectView, to ProjectView) ScaffoldCommand {
	return ScaffoldCommand{Command: "dotnet add ./" + from.Path + " reference ./" + to.Path}
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
