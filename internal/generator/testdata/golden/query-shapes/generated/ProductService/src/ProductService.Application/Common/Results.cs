namespace ProductService.Application.Common;

public sealed record PagedResult<T>(IReadOnlyList<T> Items, int Page, int PageSize, int TotalCount);
public enum MutationPreparationStatus { Prepared, InvalidToken }
public sealed record EntitySnapshot<T>(T Entity, string ConcurrencyToken);
