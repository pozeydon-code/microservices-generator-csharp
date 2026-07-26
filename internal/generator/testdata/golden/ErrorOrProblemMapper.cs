using ErrorOr;
using Microsoft.AspNetCore.Mvc;

namespace ProductService.WebApi.Common;

public static class ErrorOrProblemMapper
{
    public static ActionResult<T> ToActionResult<T>(ControllerBase controller, IReadOnlyList<Error> errors)
    {
        if (errors.Any(error => error.Type == ErrorType.Validation))
        {
            return controller.BadRequest(ToValidationProblem(errors.Where(error => error.Type == ErrorType.Validation)));
        }

        var error = errors[0];
        return error.Type switch
        {
            ErrorType.NotFound => controller.NotFound(),
            ErrorType.Conflict => controller.Conflict(),
            ErrorType.Unauthorized => controller.Unauthorized(),
            ErrorType.Forbidden => controller.Forbid(),
            _ => controller.StatusCode(StatusCodes.Status500InternalServerError)
        };
    }

    private static ValidationProblemDetails ToValidationProblem(IEnumerable<Error> errors) => new(
        errors
            .GroupBy(FieldFor)
            .OrderBy(group => group.Key, StringComparer.Ordinal)
            .ToDictionary(group => group.Key, group => group.OrderBy(error => error.Code, StringComparer.Ordinal).Select(error => error.Code + ": " + error.Description).ToArray(), StringComparer.Ordinal))
    {
        Title = "Validation failed",
        Status = StatusCodes.Status400BadRequest
    };

    private static string FieldFor(Error error) => error.Metadata is not null && error.Metadata.TryGetValue("field", out var field)
        ? field.ToString() ?? string.Empty
        : string.Empty;
}
