namespace ProductService.Application.Categories.Dtos;

public sealed record CreateCategoryRequest
{
    public string Name { get; init; } = string.Empty;
    public Guid CategoryId { get; init; }
}
