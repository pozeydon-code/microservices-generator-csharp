namespace OrderingService.Domain.Entities;



public sealed class OrderItemState
{
    public required string Sku { get; init; }
    public required Guid OrderId { get; init; }
}

public sealed class OrderItem
{
    private OrderItem() { }

    public Guid Id { get; private set; }
    public string Sku { get; private set; } = string.Empty;
    public Guid OrderId { get; private set; }
    public Order Order { get; private set; } = null!;

    public byte[] RowVersion { get; private set; } = [];

    public static OrderItem Create(OrderItemState state) => new()
    {
        Id = Guid.NewGuid(),
        Sku = state.Sku,
        OrderId = state.OrderId,
    };

    public void Update(OrderItemState state)
    {
        Sku = state.Sku;
        OrderId = state.OrderId;
    }
}
