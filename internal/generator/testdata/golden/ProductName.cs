using System.Globalization;
using System.Text.RegularExpressions;
using ErrorOr;

namespace ProductService.Domain.ValueObjects;

public sealed class ProductName : IEquatable<ProductName>
{
    private static readonly Regex Pattern = new("^[A-Za-z0-9 .'-]+$", RegexOptions.CultureInvariant | RegexOptions.NonBacktracking);

    private ProductName(string value) => Value = value;

    public string Value { get; }

    public static ErrorOr<ProductName> Create(string? value, string? field = null)
    {
        if (value is null) return Error.Validation(code: "ProductName.Null", description: "ProductName must not be null.", metadata: new Dictionary<string, object> { ["field"] = field ?? "value" });
        if (string.IsNullOrWhiteSpace(value)) return Error.Validation(code: "ProductName.Required", description: "ProductName is required.", metadata: new Dictionary<string, object> { ["field"] = field ?? "value" });
        if (value!.Length < 3) return Error.Validation(code: "ProductName.MinLength", description: "ProductName must be at least 3 characters.", metadata: new Dictionary<string, object> { ["field"] = field ?? "value" });
        if (value.Length > 100) return Error.Validation(code: "ProductName.MaxLength", description: "ProductName must be at most 100 characters.", metadata: new Dictionary<string, object> { ["field"] = field ?? "value" });
        if (!Pattern.IsMatch(value)) return Error.Validation(code: "ProductName.Pattern", description: "ProductName has an invalid format.", metadata: new Dictionary<string, object> { ["field"] = field ?? "value" });
        return new ProductName(value);
    }

    public static ProductName Rehydrate(string value)
    {
        var result = Create(value);
        return result.IsError
            ? throw new InvalidOperationException($"Failed to rehydrate ProductName: {string.Join(", ", result.Errors.Select(e => e.Code))}")
            : result.Value;
    }

    public bool Equals(ProductName? other) => other is not null && EqualityComparer<string>.Default.Equals(Value, other.Value);
    public override bool Equals(object? obj) => Equals(obj as ProductName);
    public override int GetHashCode() => EqualityComparer<string>.Default.GetHashCode(Value);
    public override string ToString() => Convert.ToString(Value, CultureInfo.InvariantCulture) ?? string.Empty;
}
