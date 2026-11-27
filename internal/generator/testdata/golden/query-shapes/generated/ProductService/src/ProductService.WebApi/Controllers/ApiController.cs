using ProductService.WebApi.Common.Errors;
using ErrorOr;
using Microsoft.AspNetCore.Mvc;
using Microsoft.AspNetCore.Mvc.ModelBinding;

namespace ProductService.WebApi.Controllers;

[ApiController]
public abstract class ApiController : ControllerBase
{
    protected ActionResult Problem(IReadOnlyList<Error> errors)
    {
        if (errors.Count == 0)
        {
            return Problem();
        }

        HttpContext.Items[HttpContextItemKeys.Errors] = errors;

        if (errors.Any(error => error.Type == ErrorType.Validation))
        {
            var modelState = new ModelStateDictionary();
            foreach (var error in errors.Where(error => error.Type == ErrorType.Validation))
            {
                modelState.AddModelError(FieldFor(error), error.Code + ": " + error.Description);
            }

            return ValidationProblem(modelState);
        }

        var firstError = errors[0];
        return Problem(statusCode: StatusCodeFor(firstError.Type), title: firstError.Description);
    }

    private static string FieldFor(Error error) => error.Metadata is not null && error.Metadata.TryGetValue("field", out var field)
        ? field?.ToString() ?? error.Code
        : error.Code;

    private static int StatusCodeFor(ErrorType errorType) => errorType switch
    {
        ErrorType.Validation => StatusCodes.Status400BadRequest,
        ErrorType.Conflict => StatusCodes.Status409Conflict,
        ErrorType.NotFound => StatusCodes.Status404NotFound,
        ErrorType.Unauthorized => StatusCodes.Status401Unauthorized,
        ErrorType.Forbidden => StatusCodes.Status403Forbidden,
        _ => StatusCodes.Status500InternalServerError
    };
}
