namespace ProductService.Application.Products.Dtos;

public sealed record ProductDto(
    Guid Id,
    string Name,
    decimal Price,
    string ConcurrencyToken);


public sealed record ProductSummaryDto(
    Guid Id,
    string Name,
    decimal Price,
    string ConcurrencyToken);


public sealed record ProductDetailDto(
    Guid Id,
    string Name,
    decimal Price,
    string ConcurrencyToken);

