namespace ProductService.Domain.Common;

public sealed record DomainError(string Code, string Message, string? Field = null);

public sealed class DomainResult<T>
{
    internal DomainResult(T? value, IReadOnlyList<DomainError> errors)
    {
        if (errors.Count == 0 && value is null) throw new ArgumentNullException(nameof(value), "Successful domain results require a value.");
        if (errors.Count > 0 && value is not null) throw new ArgumentException("Failed domain results must not carry a value.", nameof(value));
        if (errors.Count == 0 != (value is not null)) throw new ArgumentException("Domain result state is inconsistent.", nameof(errors));
        Value = value;
        Errors = errors;
    }

    public T? Value { get; }
    public IReadOnlyList<DomainError> Errors { get; }
    public bool IsSuccess => Errors.Count == 0;

}

public static class DomainResult
{
    public static DomainResult<T> Success<T>(T value) => value is null ? throw new ArgumentNullException(nameof(value)) : new(value, []);
    public static DomainResult<T> Failure<T>(params DomainError[] errors) => errors.Length == 0 ? throw new ArgumentException("At least one error is required.", nameof(errors)) : new(default, errors);
}
