using ProductService.Application.Common;
using ProductService.Application.Features.Products;
using ProductService.Application.Features.Products.Create;
using ProductService.Application.Features.Products.Delete;
using ProductService.Application.Features.Products.GetById;
using ProductService.Application.Features.Products.List;
using ProductService.Application.Features.Products.Update;
using ProductService.WebApi.Common;
using ErrorOr;
using MediatR;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;

namespace ProductService.WebApi.Controllers.Products;

[ApiController]
[Authorize]
[Route("products")]
public sealed class ProductController(ISender sender) : ControllerBase
{
    [HttpGet(Name = "ListProducts")]
    public async Task<ActionResult<PagedResult<ProductDto>>> List([FromQuery] int? page, [FromQuery] int? pageSize, CancellationToken cancellationToken)
    {
        try
        {
            var result = await sender.Send(new ListProductQuery(page, pageSize), cancellationToken);
            return result.IsError
                ? ErrorOrProblemMapper.ToActionResult<PagedResult<ProductDto>>(this, result.Errors)
                : Ok(result.Value);
        }
        catch (ArgumentOutOfRangeException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpGet("{id:guid}", Name = "GetProductById")]
    public async Task<ActionResult<ProductDto>> Get(Guid id, CancellationToken cancellationToken)
    {
        var result = await sender.Send(new GetProductByIdQuery(id), cancellationToken);
        return result.IsError
            ? ErrorOrProblemMapper.ToActionResult<ProductDto>(this, result.Errors)
            : Ok(result.Value);
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
        var updated = await sender.Send(new UpdateProductCommand
        {
            Id = id,
            IsActive = request.IsActive,
            Name = request.Name,
            Price = request.Price,
            ConcurrencyToken = request.ConcurrencyToken,
        }, cancellationToken);
        return updated.IsError
            ? ErrorOrProblemMapper.ToActionResult<ProductDto>(this, updated.Errors)
            : Ok(updated.Value);
    }

    [HttpDelete("{id:guid}", Name = "DeleteProduct")]
    public async Task<IActionResult> Delete(Guid id, [FromQuery] string? concurrencyToken, CancellationToken cancellationToken)
    {
        var deleted = await sender.Send(new DeleteProductCommand(id, concurrencyToken ?? string.Empty), cancellationToken);
        if (deleted.IsError)
        {
            return ErrorOrProblemMapper.ToActionResult<Deleted>(this, deleted.Errors).Result!;
        }
        return NoContent();
    }
}
