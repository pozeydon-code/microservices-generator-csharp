namespace ProductService.Application.Categories.Dtos;

public sealed record CategoryDto(
    Guid Id,
    string Name,
    Guid CategoryId,
    string ConcurrencyToken);


public sealed record CategorySummaryDto(
    Guid Id,
    string Name,
    Guid CategoryId,
    string ConcurrencyToken);


public sealed record CategoryDetailDto(
    Guid Id,
    string Name,
    Guid CategoryId,
    string ConcurrencyToken);

