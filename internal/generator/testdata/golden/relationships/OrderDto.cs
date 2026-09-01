namespace OrderingService.Application.Orders.Dtos;

public sealed record OrderDto(
    Guid Id,
    string Number,
    Guid? CustomerId,
    string ConcurrencyToken);
