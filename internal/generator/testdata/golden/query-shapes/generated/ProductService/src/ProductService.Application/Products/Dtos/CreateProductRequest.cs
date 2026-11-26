namespace ProductService.Application.Products.Dtos;

public sealed record CreateProductRequest
{
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
}
