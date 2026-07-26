using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Application.Features.Products.Create;
using ProductService.WebApi.Common;
using MediatR;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace ProductService.WebApi.Controllers.Products;

[ApiController]
[Authorize]
[Route("products")]
public sealed class ProductController(IProductUseCases useCases, ISender sender) : ControllerBase
{
    [HttpGet(Name = "ListProducts")]
    public async Task<ActionResult<PagedResult<ProductDto>>> List([FromQuery] int? page, [FromQuery] int? pageSize, CancellationToken cancellationToken)
    {
        try
        {
            return Ok(await useCases.ListAsync(new PageRequest(page, pageSize), cancellationToken));
        }
        catch (ArgumentOutOfRangeException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpGet("{id:guid}", Name = "GetProductById")]
    public async Task<ActionResult<ProductDto>> Get(Guid id, CancellationToken cancellationToken)
    {
        var item = await useCases.GetByIdAsync(id, cancellationToken);
        return item is null ? NotFound() : Ok(item);
    }

    [HttpPost(Name = "CreateProduct")]
    public async Task<ActionResult<ProductDto>> Create([FromBody] CreateProductRequest request, CancellationToken cancellationToken)
    {
        var created = await sender.Send(new CreateProductCommand
        {
            IsActive = request.IsActive,
            Name = request.Name,
            Price = request.Price,
        }, cancellationToken);
        if (created.IsError)
        {
            return ErrorOrProblemMapper.ToActionResult<ProductDto>(this, created.Errors);
        }
        return CreatedAtRoute("GetProductById", new { id = created.Value.Id }, created.Value);
    }

    [HttpPut("{id:guid}", Name = "UpdateProduct")]
    public async Task<ActionResult<ProductDto>> Update(Guid id, [FromBody] UpdateProductRequest request, CancellationToken cancellationToken)
    {
        var updated = await useCases.UpdateAsync(id, request, cancellationToken);
        return updated.Status switch
        {
            MutationResultStatus.NotFound => NotFound(),
            MutationResultStatus.Conflict => Conflict(),
            MutationResultStatus.InvalidToken => BadRequest(new { error = "Invalid concurrency token." }),
            MutationResultStatus.ValidationFailed => BadRequest(ValidationProblemMapper.ToProblem(updated.Validation!)),
            MutationResultStatus.Updated => Ok(updated.Value),
            _ => throw new InvalidOperationException($"Unexpected update result {updated.Status}.")
        };
    }

    [HttpDelete("{id:guid}", Name = "DeleteProduct")]
    public async Task<IActionResult> Delete(Guid id, [FromQuery] string? concurrencyToken, CancellationToken cancellationToken)
    {
        var deleted = await useCases.DeleteAsync(id, concurrencyToken ?? string.Empty, cancellationToken);
        return deleted.Status switch
        {
            MutationResultStatus.NotFound => NotFound(),
            MutationResultStatus.Conflict => Conflict(),
            MutationResultStatus.InvalidToken => BadRequest(new { error = "Invalid concurrency token." }),
            MutationResultStatus.Deleted => NoContent(),
            _ => throw new InvalidOperationException($"Unexpected delete result {deleted.Status}.")
        };
    }
}
