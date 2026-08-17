using ProductService.WebApi.Common.Errors;
using Microsoft.AspNetCore.Mvc.Infrastructure;

namespace ProductService.WebApi;

public static class DependencyInjection
{
    public static IServiceCollection AddPresentation(this IServiceCollection services)
    {
        services.AddControllers();
        services.AddProblemDetails();
        services.AddSingleton<ProblemDetailsFactory, ApiProblemDetailsFactory>();

        return services;
    }
}
