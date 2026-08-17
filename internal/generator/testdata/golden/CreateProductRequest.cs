namespace ProductService.Application.Products.Dtos;

public sealed record CreateProductRequest
{
    public bool IsActive { get; init; }
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
}
