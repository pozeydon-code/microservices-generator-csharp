using ErrorOr;
using MediatR;
using OrderingService.Application.OrderItems.Dtos;

namespace OrderingService.Application.OrderItems.Commands.Create;

public sealed record CreateOrderItemCommand : IRequest<ErrorOr<OrderItemDto>>
{
    public string Sku { get; init; } = string.Empty;
    public Guid OrderId { get; init; }
}
