namespace ProductService.Domain.Entities;



public sealed class ProductState
{
    public required string Name { get; init; }
    public required decimal Price { get; init; }
}

public sealed class Product
{
    private Product() { }

    public Guid Id { get; private set; }
    public string Name { get; private set; } = string.Empty;
    public decimal Price { get; private set; }
    public ICollection<Category> Categorys { get; private set; } = [];

    public byte[] RowVersion { get; private set; } = [];

    public static Product Create(ProductState state) => new()
    {
        Id = Guid.NewGuid(),
        Name = state.Name,
        Price = state.Price,
    };

    public void Update(ProductState state)
    {
        Name = state.Name;
        Price = state.Price;
    }
}
