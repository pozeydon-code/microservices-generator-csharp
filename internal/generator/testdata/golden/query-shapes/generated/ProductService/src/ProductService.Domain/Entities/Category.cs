namespace ProductService.Domain.Entities;



public sealed class CategoryState
{
    public required string Name { get; init; }
    public required Guid CategoryId { get; init; }
}

public sealed class Category
{
    private Category() { }

    public Guid Id { get; private set; }
    public string Name { get; private set; } = string.Empty;
    public Guid CategoryId { get; private set; }
    public Product Product { get; private set; } = null!;

    public byte[] RowVersion { get; private set; } = [];

    public static Category Create(CategoryState state) => new()
    {
        Id = Guid.NewGuid(),
        Name = state.Name,
        CategoryId = state.CategoryId,
    };

    public void Update(CategoryState state)
    {
        Name = state.Name;
        CategoryId = state.CategoryId;
    }
}
