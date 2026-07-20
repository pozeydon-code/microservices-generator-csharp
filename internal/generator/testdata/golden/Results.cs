namespace ProductService.Application.Common;

public enum MutationResultStatus { Created, Updated, Deleted, NotFound, Conflict, InvalidToken, ValidationFailed }
public sealed record MutationResult<T>(MutationResultStatus Status, T? Value = default)
{
    public static MutationResult<T> Deleted(T value) => new(MutationResultStatus.Deleted, value);
    public static MutationResult<T> NotFound() => new(MutationResultStatus.NotFound);
    public static MutationResult<T> Conflict() => new(MutationResultStatus.Conflict);
    public static MutationResult<T> InvalidToken() => new(MutationResultStatus.InvalidToken);
}
public sealed record PagedResult<T>(IReadOnlyList<T> Items, int Page, int PageSize, int TotalCount);
public sealed record PageRequest(int? Page, int? PageSize);
public enum SaveResultStatus { Saved, Conflict, InvalidToken }
public sealed record EntitySnapshot<T>(T Entity, string ConcurrencyToken);
public sealed record ValidationIssue(string Field, string Code, string Message);
public sealed record ValidationOutcome(IReadOnlyList<ValidationIssue> Issues);
public sealed record MutationValidationResult<T>(MutationResultStatus Status, T? Value = default, ValidationOutcome? Validation = null)
{
    public static MutationValidationResult<T> Created(T value) => new(MutationResultStatus.Created, value);
    public static MutationValidationResult<T> Updated(T value) => new(MutationResultStatus.Updated, value);
    public static MutationValidationResult<T> NotFound() => new(MutationResultStatus.NotFound);
    public static MutationValidationResult<T> Conflict() => new(MutationResultStatus.Conflict);
    public static MutationValidationResult<T> InvalidToken() => new(MutationResultStatus.InvalidToken);
    public static MutationValidationResult<T> ValidationFailed(ValidationOutcome validation) => new(MutationResultStatus.ValidationFailed, Validation: validation);
}
