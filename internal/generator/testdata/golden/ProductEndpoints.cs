using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Api.Common;
using Microsoft.AspNetCore.Builder;
using Microsoft.AspNetCore.Http;
using Microsoft.AspNetCore.Routing;

namespace ProductService.Api.Features.Products;

public static class ProductEndpoints
{
    public static IEndpointRouteBuilder MapProductEndpoints(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/products").RequireAuthorization().WithTags("Product");

        group.MapGet("/", async (int? page, int? pageSize, IProductUseCases useCases, CancellationToken cancellationToken) =>
        {
            try
            {
                return Results.Ok(await useCases.ListAsync(new PageRequest(page, pageSize), cancellationToken));
            }
            catch (ArgumentOutOfRangeException ex)
            {
                return Results.BadRequest(new { error = ex.Message });
            }
        })
            .WithName("ListProducts");

        group.MapGet("/{id:guid}", async (Guid id, IProductUseCases useCases, CancellationToken cancellationToken) =>
        {
            var item = await useCases.GetByIdAsync(id, cancellationToken);
            return item is null ? Results.NotFound() : Results.Ok(item);
        }).WithName("GetProductById");

        group.MapPost("/", async (CreateProductRequest request, IProductUseCases useCases, CancellationToken cancellationToken) =>
        {
            var created = await useCases.CreateAsync(request, cancellationToken);
            return created.Status switch
            {
                MutationResultStatus.Created => Results.CreatedAtRoute("GetProductById", new { id = created.Value!.Id }, created.Value),
                MutationResultStatus.ValidationFailed => ValidationProblemMapper.ToProblem(created.Validation!),
                _ => throw new InvalidOperationException($"Unexpected create result {created.Status}.")
            };
        }).WithName("CreateProduct");

        group.MapPut("/{id:guid}", async (Guid id, UpdateProductRequest request, IProductUseCases useCases, CancellationToken cancellationToken) =>
        {
            var updated = await useCases.UpdateAsync(id, request, cancellationToken);
            return updated.Status switch
            {
                MutationResultStatus.NotFound => Results.NotFound(),
                MutationResultStatus.Conflict => Results.Conflict(),
                MutationResultStatus.InvalidToken => Results.BadRequest(new { error = "Invalid concurrency token." }),
                MutationResultStatus.ValidationFailed => ValidationProblemMapper.ToProblem(updated.Validation!),
                MutationResultStatus.Updated => Results.Ok(updated.Value),
                _ => throw new InvalidOperationException($"Unexpected update result {updated.Status}.")
            };
        }).WithName("UpdateProduct");

        group.MapDelete("/{id:guid}", async (Guid id, string concurrencyToken, IProductUseCases useCases, CancellationToken cancellationToken) =>
        {
            var deleted = await useCases.DeleteAsync(id, concurrencyToken, cancellationToken);
            return deleted.Status switch
            {
                MutationResultStatus.NotFound => Results.NotFound(),
                MutationResultStatus.Conflict => Results.Conflict(),
                MutationResultStatus.InvalidToken => Results.BadRequest(new { error = "Invalid concurrency token." }),
                MutationResultStatus.Deleted => Results.NoContent(),
                _ => throw new InvalidOperationException($"Unexpected delete result {deleted.Status}.")
            };
        }).WithName("DeleteProduct");

        return app;
    }
}
