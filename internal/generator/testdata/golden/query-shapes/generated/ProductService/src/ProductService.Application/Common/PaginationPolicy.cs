namespace ProductService.Application.Common;

public static class PaginationPolicy
{
    public const int DefaultPage = 1;
    public const int DefaultPageSize = 20;
    public const int MaxPageSize = 100;

    public static (int Page, int PageSize, int Offset) Normalize(int? page, int? pageSize)
    {
        var normalizedPage = page.GetValueOrDefault(DefaultPage);
        var normalizedPageSize = pageSize.GetValueOrDefault(DefaultPageSize);
        if (normalizedPage < 1) throw new ArgumentOutOfRangeException(nameof(page), "Page must be greater than or equal to 1.");
        if (normalizedPageSize < 1 || normalizedPageSize > MaxPageSize) throw new ArgumentOutOfRangeException(nameof(pageSize), $"Page size must be between 1 and {MaxPageSize}.");
        var offset = ((long)normalizedPage - 1L) * normalizedPageSize;
        if (offset > int.MaxValue) throw new ArgumentOutOfRangeException(nameof(page), "Page offset is too large for the supported query range.");
        return (normalizedPage, normalizedPageSize, checked((int)offset));
    }
}
