namespace ProductService.Application.Categories.Dtos;

public sealed record UpdateCategoryRequest
{
    public string Name { get; init; } = string.Empty;
    public Guid CategoryId { get; init; }
    public string ConcurrencyToken { get; init; } = string.Empty;
}
