namespace OrderingService.Domain.Entities;



public sealed class OrderState
{
    public required string Number { get; init; }
    public required Guid? CustomerId { get; init; }
}

public sealed class Order
{
    private Order() { }

    public Guid Id { get; private set; }
    public string Number { get; private set; } = string.Empty;
    public Guid? CustomerId { get; private set; }
    public Customer? Customer { get; private set; }
    public ICollection<OrderItem> Items { get; private set; } = [];

    public byte[] RowVersion { get; private set; } = [];

    public static Order Create(OrderState state) => new()
    {
        Id = Guid.NewGuid(),
        Number = state.Number,
        CustomerId = state.CustomerId,
    };

    public void Update(OrderState state)
    {
        Number = state.Number;
        CustomerId = state.CustomerId;
    }
}
