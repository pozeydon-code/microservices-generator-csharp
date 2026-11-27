using ProductService.Application.Common;
using ProductService.Application.Categories.Commands.Create;
using ProductService.Application.Categories.Commands.Delete;
using ProductService.Application.Categories.Commands.Update;
using ProductService.Application.Categories.Dtos;
using ProductService.Application.Categories.Queries.GetById;
using ProductService.Application.Categories.Queries.List;
using ErrorOr;
using MediatR;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using ProductService.WebApi.Controllers;

namespace ProductService.WebApi.Controllers.Categories;

[ApiController]
[Authorize]
[Route("categories")]
public sealed class CategoryController(ISender sender) : ApiController
{
    [HttpGet(Name = "ListCategories")]
    public async Task<ActionResult<PagedResult<CategoryDto>>> List([FromQuery] int? page, [FromQuery] int? pageSize, [FromQuery] string? shape, CancellationToken cancellationToken)
    {
        if (shape is not null && !IsValidShape(shape))
        {
            return BadRequest(new { error = $"Invalid shape: {shape}. Valid shapes are: {string.Join(", ", ValidShapes())}" });
        }
        try
        {
            var result = await sender.Send(new ListCategoryQuery(page, pageSize, shape), cancellationToken);
            return result.IsError
                ? Problem(result.Errors)
                : Ok(result.Value);
        }
        catch (ArgumentOutOfRangeException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpGet("{id:guid}", Name = "GetCategoryById")]
    public async Task<ActionResult<CategoryDto>> Get(Guid id, [FromQuery] string? shape, CancellationToken cancellationToken)
    {
        if (shape is not null && !IsValidShape(shape))
        {
            return BadRequest(new { error = $"Invalid shape: {shape}. Valid shapes are: {string.Join(", ", ValidShapes())}" });
        }
        var result = await sender.Send(new GetCategoryByIdQuery(id, shape), cancellationToken);
        return result.IsError
            ? Problem(result.Errors)
            : Ok(result.Value);
    }

    [HttpPost(Name = "CreateCategory")]
    public async Task<ActionResult<CategoryDto>> Create([FromBody] CreateCategoryRequest request, CancellationToken cancellationToken)
    {
        var created = await sender.Send(new CreateCategoryCommand
        {
            Name = request.Name,
            CategoryId = request.CategoryId,
        }, cancellationToken);
        if (created.IsError)
        {
            return Problem(created.Errors);
        }
        return CreatedAtRoute("GetCategoryById", new { id = created.Value.Id }, created.Value);
    }

    [HttpPut("{id:guid}", Name = "UpdateCategory")]
    public async Task<ActionResult<CategoryDto>> Update(Guid id, [FromBody] UpdateCategoryRequest request, CancellationToken cancellationToken)
    {
        var updated = await sender.Send(new UpdateCategoryCommand
        {
            Id = id,
            Name = request.Name,
            CategoryId = request.CategoryId,
            ConcurrencyToken = request.ConcurrencyToken,
        }, cancellationToken);
        return updated.IsError
            ? Problem(updated.Errors)
            : Ok(updated.Value);
    }

    [HttpDelete("{id:guid}", Name = "DeleteCategory")]
    public async Task<IActionResult> Delete(Guid id, [FromQuery] string? concurrencyToken, CancellationToken cancellationToken)
    {
        var deleted = await sender.Send(new DeleteCategoryCommand(id, concurrencyToken ?? string.Empty), cancellationToken);
        if (deleted.IsError)
        {
            return Problem(deleted.Errors);
        }
        return NoContent();
    }

    private static bool IsValidShape(string shape)
    {
        return shape switch
        {
            "summary" => true,
            "detail" => true,
            _ => false
        };
    }

    private static string[] ValidShapes()
    {
        return new[]
        {
            "summary",
            "detail",
        };
    }
}
