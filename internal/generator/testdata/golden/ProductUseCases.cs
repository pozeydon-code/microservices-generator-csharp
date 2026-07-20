using ProductService.Application.Common;
using ProductService.Domain.Features.Products;
using ProductService.Domain.Shared.ValueObjects;


namespace ProductService.Application.Features.Products;

public sealed class ProductUseCases(IProductRepository repository) : IProductUseCases
{
    public async Task<PagedResult<ProductDto>> ListAsync(PageRequest request, CancellationToken cancellationToken)
    {
        var normalized = PaginationPolicy.Normalize(request.Page, request.PageSize);
        var result = await repository.ListAsync(normalized.Offset, normalized.PageSize, cancellationToken);
        return new PagedResult<ProductDto>(result.Items.Select(ToDto).ToList(), normalized.Page, normalized.PageSize, result.TotalCount);
    }

    public async Task<ProductDto?> GetByIdAsync(Guid id, CancellationToken cancellationToken)
    {
        var snapshot = await repository.GetByIdAsync(id, cancellationToken);
        return snapshot is null ? null : ToDto(snapshot);
    }

    public async Task<MutationValidationResult<ProductDto>> CreateAsync(CreateProductRequest request, CancellationToken cancellationToken)
    {
        var state = CreateState(request);
        if (state.Validation.Issues.Count > 0)
        {
            return MutationValidationResult<ProductDto>.ValidationFailed(state.Validation);
        }
        var entity = Product.Create(state.State!);
        var snapshot = await repository.AddAsync(entity, cancellationToken);
        return MutationValidationResult<ProductDto>.Created(ToDto(snapshot));
    }

    public async Task<MutationValidationResult<ProductDto>> UpdateAsync(Guid id, UpdateProductRequest request, CancellationToken cancellationToken)
    {
        var state = CreateState(request);
        if (state.Validation.Issues.Count > 0)
        {
            return MutationValidationResult<ProductDto>.ValidationFailed(state.Validation);
        }
        if (string.IsNullOrWhiteSpace(request.ConcurrencyToken))
        {
            return MutationValidationResult<ProductDto>.InvalidToken();
        }
        var snapshot = await repository.GetByIdAsync(id, cancellationToken);
        if (snapshot is null)
        {
            return MutationValidationResult<ProductDto>.NotFound();
        }
        snapshot.Entity.Update(state.State!);
        var status = await repository.UpdateAsync(snapshot.Entity, request.ConcurrencyToken, cancellationToken);
        if (status == SaveResultStatus.Conflict)
        {
            return MutationValidationResult<ProductDto>.Conflict();
        }
        if (status == SaveResultStatus.InvalidToken)
        {
            return MutationValidationResult<ProductDto>.InvalidToken();
        }
        var updated = await repository.GetByIdAsync(id, cancellationToken);
        return MutationValidationResult<ProductDto>.Updated(ToDto(updated ?? snapshot));
    }

    public async Task<MutationResult<bool>> DeleteAsync(Guid id, string concurrencyToken, CancellationToken cancellationToken)
    {
        var snapshot = await repository.GetByIdAsync(id, cancellationToken);
        if (snapshot is null)
        {
            return new MutationResult<bool>(MutationResultStatus.NotFound);
        }
        if (string.IsNullOrWhiteSpace(concurrencyToken))
        {
            return new MutationResult<bool>(MutationResultStatus.InvalidToken);
        }
        var status = await repository.DeleteAsync(snapshot.Entity, concurrencyToken, cancellationToken);
        if (status == SaveResultStatus.Conflict)
        {
            return new MutationResult<bool>(MutationResultStatus.Conflict);
        }
        if (status == SaveResultStatus.InvalidToken)
        {
            return new MutationResult<bool>(MutationResultStatus.InvalidToken);
        }
        return new MutationResult<bool>(MutationResultStatus.Deleted, true);
    }

    private static ProductDto ToDto(EntitySnapshot<Product> snapshot) => new(
        snapshot.Entity.Id,
        snapshot.Entity.IsActive,
        snapshot.Entity.Name.Value,
        snapshot.Entity.Price.Value,
        snapshot.ConcurrencyToken);

    private static (ProductState? State, ValidationOutcome Validation) CreateState(CreateProductRequest request) => CreateStateCore(ProductPrimitiveInput.From(request));
    private static (ProductState? State, ValidationOutcome Validation) CreateState(UpdateProductRequest request) => CreateStateCore(ProductPrimitiveInput.From(request));

    private static (ProductState? State, ValidationOutcome Validation) CreateStateCore(ProductPrimitiveInput input)
    {
        List<ValidationIssue> issues = [];
        var nameResult = ProductName.Create(input.Name, "name");
        issues.AddRange(nameResult.Errors.Select(error => new ValidationIssue(error.Field ?? "name", error.Code, error.Message)));
        var priceResult = ProductPrice.Create(input.Price, "price");
        issues.AddRange(priceResult.Errors.Select(error => new ValidationIssue(error.Field ?? "price", error.Code, error.Message)));
        if (issues.Count > 0)
        {
            return (null, new ValidationOutcome(issues.OrderBy(issue => issue.Field, StringComparer.Ordinal).ThenBy(issue => issue.Code, StringComparer.Ordinal).ToArray()));
        }
        return (new ProductState
        {
            IsActive = input.IsActive,
            Name = nameResult.Value!,
            Price = priceResult.Value!,
        }, new ValidationOutcome([]));
    }
}
