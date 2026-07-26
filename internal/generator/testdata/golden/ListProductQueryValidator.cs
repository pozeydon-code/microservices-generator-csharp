using FluentValidation;
using FluentValidation.Results;
using ProductService.Application.Common;

namespace ProductService.Application.Features.Products.List;

public sealed class ListProductQueryValidator : AbstractValidator<ListProductQuery>
{
    public ListProductQueryValidator()
    {
        RuleFor(query => query.Page).Custom((value, context) =>
        {
            if (value is < 1)
            {
                context.AddFailure(new ValidationFailure(context.PropertyPath, "Page must be greater than or equal to 1.") { ErrorCode = "Pagination.Page" });
            }
        });
        RuleFor(query => query.PageSize).Custom((value, context) =>
        {
            if (value is < 1 or > PaginationPolicy.MaxPageSize)
            {
                context.AddFailure(new ValidationFailure(context.PropertyPath, $"Page size must be between 1 and {PaginationPolicy.MaxPageSize}.") { ErrorCode = "Pagination.PageSize" });
            }
        });
        RuleFor(query => query).Custom((query, context) =>
        {
            var page = query.Page.GetValueOrDefault(PaginationPolicy.DefaultPage);
            var pageSize = query.PageSize.GetValueOrDefault(PaginationPolicy.DefaultPageSize);
            if (page < 1 || pageSize < 1 || pageSize > PaginationPolicy.MaxPageSize)
            {
                return;
            }
            var offset = ((long)page - 1L) * pageSize;
            if (offset > int.MaxValue)
            {
                context.AddFailure(new ValidationFailure(nameof(query.Page), "Page offset is too large for the supported query range.") { ErrorCode = "Pagination.PageOffset" });
            }
        });
    }
}
