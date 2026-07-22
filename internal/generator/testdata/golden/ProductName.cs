using System.Globalization;
using System.Text.RegularExpressions;
using ProductService.Domain.Common;

namespace ProductService.Domain.Common.ValueObjects;

public sealed class ProductName : IEquatable<ProductName>
{
    private static readonly Regex Pattern = new("^[A-Za-z0-9 .'-]+$", RegexOptions.CultureInvariant | RegexOptions.NonBacktracking);

    private ProductName(string value) => Value = value;

    public string Value { get; }

    public static DomainResult<ProductName> Create(string? value, string? field = null)
    {
        if (value is null) return DomainResult.Failure<ProductName>(new DomainError("ProductName.Null", "ProductName must not be null.", field));
        if (string.IsNullOrWhiteSpace(value)) return DomainResult.Failure<ProductName>(new DomainError("ProductName.Required", "ProductName is required.", field));
        if (value!.Length < 3) return DomainResult.Failure<ProductName>(new DomainError("ProductName.MinLength", "ProductName must be at least 3 characters.", field));
        if (value.Length > 100) return DomainResult.Failure<ProductName>(new DomainError("ProductName.MaxLength", "ProductName must be at most 100 characters.", field));
        if (!Pattern.IsMatch(value)) return DomainResult.Failure<ProductName>(new DomainError("ProductName.Pattern", "ProductName has an invalid format.", field));
        return DomainResult.Success(new ProductName(value));
    }

    public static ProductName Rehydrate(string value)
    {
        var result = Create(value);
        return result.Value ?? throw new DomainReconstitutionException("ProductName", result.Errors.Select(error => error.Code).ToArray());
    }

    public bool Equals(ProductName? other) => other is not null && EqualityComparer<string>.Default.Equals(Value, other.Value);
    public override bool Equals(object? obj) => Equals(obj as ProductName);
    public override int GetHashCode() => EqualityComparer<string>.Default.GetHashCode(Value);
    public override string ToString() => Convert.ToString(Value, CultureInfo.InvariantCulture) ?? string.Empty;
}
