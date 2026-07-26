namespace ProductService.Application.Features.Products;

public sealed record ProductDto(
    Guid Id,
    bool IsActive,
    string Name,
    decimal Price,
    string ConcurrencyToken);

public sealed record CreateProductRequest
{
    public bool IsActive { get; init; }
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
}

public sealed record UpdateProductRequest
{
    public bool IsActive { get; init; }
    public string Name { get; init; } = string.Empty;
    public decimal Price { get; init; }
    public string ConcurrencyToken { get; init; } = string.Empty;
}
