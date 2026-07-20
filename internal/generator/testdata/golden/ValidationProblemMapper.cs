using ProductService.Application.Common;
using Microsoft.AspNetCore.Http;

namespace ProductService.Api.Common;

public static class ValidationProblemMapper
{
    public static IResult ToProblem(ValidationOutcome validation) => Results.ValidationProblem(
        validation.Issues
            .GroupBy(issue => issue.Field, StringComparer.Ordinal)
            .OrderBy(group => group.Key, StringComparer.Ordinal)
            .ToDictionary(group => group.Key, group => group.OrderBy(issue => issue.Code, StringComparer.Ordinal).Select(issue => issue.Code + ": " + issue.Message).ToArray(), StringComparer.Ordinal),
        title: "Validation failed",
        statusCode: StatusCodes.Status400BadRequest);
}
