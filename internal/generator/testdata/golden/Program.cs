using ProductService.Application.Features.Products;
using ProductService.Api.Features.Products;

using ProductService.Api.Health;
using ProductService.Infrastructure;
using Microsoft.AspNetCore.Authentication.JwtBearer;
using Microsoft.AspNetCore.Http.Timeouts;
using Microsoft.IdentityModel.Tokens;
using OpenTelemetry.Metrics;
using OpenTelemetry.Resources;
using OpenTelemetry.Trace;

var builder = WebApplication.CreateBuilder(args);
var resilience = ResilienceOptions.From(builder.Configuration);
var telemetry = TelemetryOptions.From(builder.Configuration);

builder.Logging.AddConsole();
builder.Services.AddProblemDetails();
builder.Services.AddHttpClient();
builder.Services.AddRequestTimeouts(options => options.DefaultPolicy = new RequestTimeoutPolicy
{
    Timeout = TimeSpan.FromSeconds(resilience.RequestTimeoutSeconds),
    TimeoutStatusCode = StatusCodes.Status503ServiceUnavailable,
    WriteTimeoutResponse = async context =>
    {
        context.Response.ContentType = "application/problem+json";
        await context.Response.WriteAsJsonAsync(new
        {
            type = "https://httpstatuses.com/503",
            title = "Service Unavailable",
            status = StatusCodes.Status503ServiceUnavailable,
            detail = "The request timed out."
        });
    }
});
builder.Services.AddOpenTelemetry()
    .ConfigureResource(resource => resource.AddService(
        serviceName: "ProductService",
        serviceVersion: telemetry.ServiceVersion)
        .AddAttributes([new KeyValuePair<string, object>("deployment.environment", telemetry.DeploymentEnvironment)]))
    .WithTracing(tracing => tracing
        .AddSource("ProductService.Infrastructure")
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddOtlpExporter())
    .WithMetrics(metrics => metrics
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddOtlpExporter());

var authority = builder.Configuration["Authentication:Authority"];
var audience = builder.Configuration["Authentication:Audience"];
if (string.IsNullOrWhiteSpace(authority))
{
    throw new InvalidOperationException("Missing authentication authority. Set Authentication:Authority or Authentication__Authority.");
}
if (string.IsNullOrWhiteSpace(audience))
{
    throw new InvalidOperationException("Missing authentication audience. Set Authentication:Audience or Authentication__Audience.");
}

builder.Services.AddAuthentication(JwtBearerDefaults.AuthenticationScheme)
    .AddJwtBearer(options =>
    {
        options.Authority = authority;
        options.Audience = audience;
        options.TokenValidationParameters = new TokenValidationParameters
        {
            ValidateIssuer = true,
            ValidateAudience = true,
            ValidateLifetime = true,
            ValidateIssuerSigningKey = true
        };
    });
builder.Services.AddAuthorization();
builder.Services.AddScoped<IProductUseCases, ProductUseCases>();

builder.Services.AddInfrastructure(builder.Configuration);

var app = builder.Build();

if (!app.Environment.IsDevelopment() && !app.Environment.IsEnvironment("Testing"))
{
    app.UseHsts();
    app.Use(async (context, next) =>
    {
        context.Response.Headers.TryAdd("Strict-Transport-Security", "max-age=2592000");
        await next(context);
    });
}

app.UseExceptionHandler();
app.UseHttpsRedirection();
app.UseRequestTimeouts();
app.UseAuthentication();
app.UseAuthorization();

app.MapHealthEndpoints();
app.MapProductEndpoints();

app.Run();

public partial class Program;

internal sealed record ResilienceOptions(int RequestTimeoutSeconds)
{
    public static ResilienceOptions From(IConfiguration configuration)
    {
        var requestTimeoutSeconds = configuration.GetValue("Resilience:RequestTimeoutSeconds", 15);
        if (requestTimeoutSeconds < 1 || requestTimeoutSeconds > 60)
        {
            throw new InvalidOperationException("Resilience:RequestTimeoutSeconds must be between 1 and 60 seconds.");
        }
        if (requestTimeoutSeconds < 10)
        {
            throw new InvalidOperationException("Resilience:RequestTimeoutSeconds must be at least 10 seconds to leave overhead above the generated SQL timeout/retry budget.");
        }
        return new ResilienceOptions(requestTimeoutSeconds);
    }
}

internal sealed record TelemetryOptions(string ServiceVersion, string DeploymentEnvironment)
{
    public static TelemetryOptions From(IConfiguration configuration)
    {
        var endpoint = Environment.GetEnvironmentVariable("OTEL_EXPORTER_OTLP_ENDPOINT");
        var disabled = string.Equals(Environment.GetEnvironmentVariable("OTEL_SDK_DISABLED"), "true", StringComparison.OrdinalIgnoreCase);
        if (string.IsNullOrWhiteSpace(endpoint) && !disabled)
        {
            throw new InvalidOperationException("Configure OTEL_EXPORTER_OTLP_ENDPOINT or explicitly set OTEL_SDK_DISABLED=true to opt out of telemetry export.");
        }
        var version = configuration["Service:Version"] ?? Environment.GetEnvironmentVariable("SERVICE_VERSION") ?? "unknown";
        var environment = configuration["Deployment:Environment"] ?? Environment.GetEnvironmentVariable("DEPLOYMENT_ENVIRONMENT") ?? configuration["ASPNETCORE_ENVIRONMENT"] ?? "unknown";
        return new TelemetryOptions(version, environment);
    }
}
