using System.Globalization;
using System.Text.RegularExpressions;
using ErrorOr;

namespace ProductService.Domain.ValueObjects;

public sealed class ProductPrice : IEquatable<ProductPrice>
{
    private ProductPrice(decimal value) => Value = value;

    public decimal Value { get; }

    public static ErrorOr<ProductPrice> Create(decimal value, string? field = null)
    {
        if (value < 0m) return Error.Validation(code: "ProductPrice.Minimum", description: "ProductPrice must be greater than or equal to 0m.", metadata: new Dictionary<string, object> { ["field"] = field ?? "value" });
        if (value > 999999.99m) return Error.Validation(code: "ProductPrice.Maximum", description: "ProductPrice must be less than or equal to 999999.99m.", metadata: new Dictionary<string, object> { ["field"] = field ?? "value" });
        return new ProductPrice(value);
    }

    public static ProductPrice Rehydrate(decimal value)
    {
        var result = Create(value);
        return result.IsError
            ? throw new InvalidOperationException($"Failed to rehydrate ProductPrice: {string.Join(", ", result.Errors.Select(e => e.Code))}")
            : result.Value;
    }

    public bool Equals(ProductPrice? other) => other is not null && EqualityComparer<decimal>.Default.Equals(Value, other.Value);
    public override bool Equals(object? obj) => Equals(obj as ProductPrice);
    public override int GetHashCode() => EqualityComparer<decimal>.Default.GetHashCode(Value);
    public override string ToString() => Convert.ToString(Value, CultureInfo.InvariantCulture) ?? string.Empty;
}
