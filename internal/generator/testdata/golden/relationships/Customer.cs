namespace OrderingService.Domain.Entities;



public sealed class CustomerState
{
    public required string Name { get; init; }
}

public sealed class Customer
{
    private Customer() { }

    public Guid Id { get; private set; }
    public string Name { get; private set; } = string.Empty;
    public ICollection<Order> Orders { get; private set; } = [];

    public byte[] RowVersion { get; private set; } = [];

    public static Customer Create(CustomerState state) => new()
    {
        Id = Guid.NewGuid(),
        Name = state.Name,
    };

    public void Update(CustomerState state)
    {
        Name = state.Name;
    }
}
