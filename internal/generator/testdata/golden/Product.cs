namespace ProductService.Domain.Features.Products;

using ProductService.Domain.Common.ValueObjects;


public sealed class ProductState
{
    public required bool IsActive { get; init; }
    public required ProductName Name { get; init; }
    public required ProductPrice Price { get; init; }
}

public sealed class Product
{
    private Product() { }

    public Guid Id { get; private set; }
    public bool IsActive { get; private set; }
    public ProductName Name { get; private set; } = null!;
    public ProductPrice Price { get; private set; } = null!;

    public static Product Create(ProductState state) => new()
    {
        Id = Guid.NewGuid(),
        IsActive = state.IsActive,
        Name = state.Name,
        Price = state.Price,
    };

    public void Update(ProductState state)
    {
        IsActive = state.IsActive;
        Name = state.Name;
        Price = state.Price;
    }
}
