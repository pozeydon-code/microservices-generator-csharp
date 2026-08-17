namespace ProductService.Application.Products.Dtos;

public sealed record UpdateProductRequest
{
    public bool IsActive { get; init; }
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
    public string ConcurrencyToken { get; init; } = string.Empty;
}
