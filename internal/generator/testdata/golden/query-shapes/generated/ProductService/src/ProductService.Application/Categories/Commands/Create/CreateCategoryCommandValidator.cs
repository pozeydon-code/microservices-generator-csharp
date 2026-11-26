using FluentValidation;
using FluentValidation.Results;


namespace ProductService.Application.Categories.Commands.Create;

public sealed class CreateCategoryCommandValidator : AbstractValidator<CreateCategoryCommand>
{
    public CreateCategoryCommandValidator()
    {
    }
}
