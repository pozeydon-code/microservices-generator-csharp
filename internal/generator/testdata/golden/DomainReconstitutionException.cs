namespace ProductService.Domain.Common;

public sealed class DomainReconstitutionException(string valueObjectType, IReadOnlyList<string> invariantCodes)
    : InvalidOperationException($"Persisted {valueObjectType} value is invalid.")
{
    public string ValueObjectType { get; } = valueObjectType;
    public IReadOnlyList<string> InvariantCodes { get; } = invariantCodes;
}
