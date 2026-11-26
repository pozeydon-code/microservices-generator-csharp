using FluentValidation;
using FluentValidation.Results;


namespace ProductService.Application.Products.Commands.Create;

public sealed class CreateProductCommandValidator : AbstractValidator<CreateProductCommand>
{
    public CreateProductCommandValidator()
    {
    }
}
