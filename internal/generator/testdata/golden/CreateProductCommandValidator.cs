using FluentValidation;
using FluentValidation.Results;
using ProductService.Domain.ValueObjects;


namespace ProductService.Application.Features.Products.Create;

public sealed class CreateProductCommandValidator : AbstractValidator<CreateProductCommand>
{
    public CreateProductCommandValidator()
    {
        RuleFor(command => command.Name).Custom((value, context) =>
        {
            var result = ProductName.Create(value, "name");
            foreach (var error in result.Errors)
            {
                context.AddFailure(new ValidationFailure(context.PropertyPath, error.Message) { ErrorCode = error.Code });
            }
        });
        RuleFor(command => command.Price).Custom((value, context) =>
        {
            var result = ProductPrice.Create(value, "price");
            foreach (var error in result.Errors)
            {
                context.AddFailure(new ValidationFailure(context.PropertyPath, error.Message) { ErrorCode = error.Code });
            }
        });
    }
}
