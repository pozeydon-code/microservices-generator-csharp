#!/usr/bin/env bash
set -euo pipefail

if (( $# > 1 )); then
  printf 'usage: %s [target-directory]\n' "$0" >&2
  exit 2
fi

target_dir="${1:-.microgen-scaffold}"
if [[ -L "$target_dir" ]]; then
  printf 'refusing symlink target: %s\n' "$target_dir" >&2
  exit 1
fi
if [[ -e "$target_dir" && ! -d "$target_dir" ]]; then
  printf 'refusing non-directory target: %s\n' "$target_dir" >&2
  exit 1
fi
if [[ -d "$target_dir" ]] && [[ -n "$(find "$target_dir" -mindepth 1 -print -quit)" ]]; then
  printf 'refusing non-empty target: %s\n' "$target_dir" >&2
  exit 1
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
mkdir -p -- "$target_dir"
cp -- "$script_dir/Directory.Packages.props" "$target_dir/Directory.Packages.props"
cp -- "$script_dir/Directory.Build.props" "$target_dir/Directory.Build.props"
cd -- "$target_dir"

dotnet new sln --name 'CommercePlatform'
dotnet new classlib --framework 'net8.0' --name 'ProductService.Domain' --output './src/ProductService/ProductService.Domain' --no-restore
dotnet new classlib --framework 'net8.0' --name 'ProductService.Application' --output './src/ProductService/ProductService.Application' --no-restore
dotnet new classlib --framework 'net8.0' --name 'ProductService.Infrastructure' --output './src/ProductService/ProductService.Infrastructure' --no-restore
dotnet new webapi --use-controllers --no-openapi --framework 'net8.0' --name 'ProductService.WebApi' --output './src/ProductService/ProductService.WebApi' --no-restore
dotnet new xunit --framework 'net8.0' --name 'ProductService.Domain.Tests' --output './tests/ProductService/ProductService.Domain.Tests' --no-restore
dotnet new xunit --framework 'net8.0' --name 'ProductService.Application.Tests' --output './tests/ProductService/ProductService.Application.Tests' --no-restore
dotnet new xunit --framework 'net8.0' --name 'ProductService.WebApi.Tests' --output './tests/ProductService/ProductService.WebApi.Tests' --no-restore
dotnet new xunit --framework 'net8.0' --name 'ProductService.Architecture.Tests' --output './tests/ProductService/ProductService.Architecture.Tests' --no-restore
dotnet new xunit --framework 'net8.0' --name 'ProductService.Infrastructure.Tests' --output './tests/ProductService/ProductService.Infrastructure.Tests' --no-restore
dotnet remove './tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' package 'coverlet.collector'
dotnet remove './tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet remove './tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' package 'xunit'
dotnet remove './tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet remove './tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' package 'coverlet.collector'
dotnet remove './tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet remove './tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' package 'xunit'
dotnet remove './tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet remove './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'coverlet.collector'
dotnet remove './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet remove './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'xunit'
dotnet remove './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet remove './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' package 'coverlet.collector'
dotnet remove './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet remove './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' package 'xunit'
dotnet remove './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet remove './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'coverlet.collector'
dotnet remove './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet remove './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'xunit'
dotnet remove './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet sln './CommercePlatform.sln' add './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'
dotnet sln './CommercePlatform.sln' add './src/ProductService/ProductService.Application/ProductService.Application.csproj'
dotnet sln './CommercePlatform.sln' add './src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj'
dotnet sln './CommercePlatform.sln' add './src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj'
dotnet sln './CommercePlatform.sln' add './tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj'
dotnet sln './CommercePlatform.sln' add './tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj'
dotnet sln './CommercePlatform.sln' add './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj'
dotnet sln './CommercePlatform.sln' add './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj'
dotnet sln './CommercePlatform.sln' add './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj'
dotnet add './src/ProductService/ProductService.Application/ProductService.Application.csproj' reference './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'
dotnet add './src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj' reference './src/ProductService/ProductService.Application/ProductService.Application.csproj'
dotnet add './src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj' reference './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'
dotnet add './src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' reference './src/ProductService/ProductService.Application/ProductService.Application.csproj'
dotnet add './src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' reference './src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj'
dotnet add './tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' reference './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'
dotnet add './tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' reference './src/ProductService/ProductService.Application/ProductService.Application.csproj'
dotnet add './tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' reference './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'
dotnet add './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' reference './src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj'
dotnet add './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' reference './src/ProductService/ProductService.Application/ProductService.Application.csproj'
dotnet add './tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' reference './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'
dotnet add './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' reference './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'
dotnet add './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' reference './src/ProductService/ProductService.Application/ProductService.Application.csproj'
dotnet add './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' reference './src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj'
dotnet add './tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' reference './src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj'
dotnet add './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' reference './src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj'
dotnet add './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' reference './src/ProductService/ProductService.Application/ProductService.Application.csproj'
dotnet add './tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' reference './src/ProductService/ProductService.Domain/ProductService.Domain.csproj'

dotnet add 'src/ProductService/ProductService.Application/ProductService.Application.csproj' package 'ErrorOr'
dotnet add 'src/ProductService/ProductService.Application/ProductService.Application.csproj' package 'FluentValidation'
dotnet add 'src/ProductService/ProductService.Application/ProductService.Application.csproj' package 'MediatR'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'Microsoft.AspNetCore.Authentication.JwtBearer'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'ErrorOr'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'FluentValidation.DependencyInjectionExtensions'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'MediatR'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'OpenTelemetry.Exporter.OpenTelemetryProtocol'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'OpenTelemetry.Extensions.Hosting'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'OpenTelemetry.Instrumentation.AspNetCore'
dotnet add 'src/ProductService/ProductService.WebApi/ProductService.WebApi.csproj' package 'OpenTelemetry.Instrumentation.Http'
dotnet add 'src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj' package 'Microsoft.EntityFrameworkCore.Design'
dotnet add 'src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj' package 'Microsoft.EntityFrameworkCore.SqlServer'
dotnet add 'src/ProductService/ProductService.Infrastructure/ProductService.Infrastructure.csproj' package 'Microsoft.Data.SqlClient'
dotnet add 'tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'Microsoft.AspNetCore.Mvc.Testing'
dotnet add 'tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'System.IdentityModel.Tokens.Jwt'
dotnet add 'tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet add 'tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' package 'xunit'
dotnet add 'tests/ProductService/ProductService.Domain.Tests/ProductService.Domain.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet add 'tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet add 'tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' package 'xunit'
dotnet add 'tests/ProductService/ProductService.Application.Tests/ProductService.Application.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet add 'tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet add 'tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'xunit'
dotnet add 'tests/ProductService/ProductService.WebApi.Tests/ProductService.WebApi.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet add 'tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet add 'tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' package 'xunit'
dotnet add 'tests/ProductService/ProductService.Architecture.Tests/ProductService.Architecture.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet add 'tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'Microsoft.NET.Test.Sdk'
dotnet add 'tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'xunit'
dotnet add 'tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'xunit.runner.visualstudio'
dotnet add 'tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'Microsoft.EntityFrameworkCore.SqlServer'
dotnet add 'tests/ProductService/ProductService.Infrastructure.Tests/ProductService.Infrastructure.Tests.csproj' package 'Microsoft.Data.SqlClient'


if grep -R -n --include='*.csproj' 'Version=' src tests; then
  printf 'central package management check failed: a generated project contains a version attribute\n' >&2
  exit 1
fi
