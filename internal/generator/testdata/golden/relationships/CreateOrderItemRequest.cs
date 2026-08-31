namespace OrderingService.Application.OrderItems.Dtos;

public sealed record CreateOrderItemRequest
{
    public string Sku { get; init; } = string.Empty;
    public Guid OrderId { get; init; }
}
