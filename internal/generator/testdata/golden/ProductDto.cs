namespace ProductService.Application.Products.Dtos;

public sealed record ProductDto(
    Guid Id,
    bool IsActive,
    string Name,
    decimal Price,
    string ConcurrencyToken);
