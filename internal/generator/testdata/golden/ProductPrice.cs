using System.Globalization;
using System.Text.RegularExpressions;
using ProductService.Domain.Common;

namespace ProductService.Domain.Common.ValueObjects;

public sealed class ProductPrice : IEquatable<ProductPrice>
{
    private ProductPrice(decimal value) => Value = value;

    public decimal Value { get; }

    public static DomainResult<ProductPrice> Create(decimal value, string? field = null)
    {
        if (value < 0m) return DomainResult.Failure<ProductPrice>(new DomainError("ProductPrice.Minimum", "ProductPrice must be greater than or equal to 0m.", field));
        if (value > 999999.99m) return DomainResult.Failure<ProductPrice>(new DomainError("ProductPrice.Maximum", "ProductPrice must be less than or equal to 999999.99m.", field));
        return DomainResult.Success(new ProductPrice(value));
    }

    public static ProductPrice Rehydrate(decimal value)
    {
        var result = Create(value);
        return result.Value ?? throw new DomainReconstitutionException("ProductPrice", result.Errors.Select(error => error.Code).ToArray());
    }

    public bool Equals(ProductPrice? other) => other is not null && EqualityComparer<decimal>.Default.Equals(Value, other.Value);
    public override bool Equals(object? obj) => Equals(obj as ProductPrice);
    public override int GetHashCode() => EqualityComparer<decimal>.Default.GetHashCode(Value);
    public override string ToString() => Convert.ToString(Value, CultureInfo.InvariantCulture) ?? string.Empty;
}
