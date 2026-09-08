using SchoolService.Application.Common;
using SchoolService.Application.StudentCourses.Commands.Create;
using SchoolService.Application.StudentCourses.Commands.Delete;
using SchoolService.Application.StudentCourses.Commands.Update;
using SchoolService.Application.StudentCourses.Dtos;
using SchoolService.Application.StudentCourses.Queries.GetById;
using SchoolService.Application.StudentCourses.Queries.List;
using ErrorOr;
using MediatR;
using Microsoft.AspNetCore.Authorization;
using Microsoft.AspNetCore.Mvc;
using SchoolService.WebApi.Controllers;

namespace SchoolService.WebApi.Controllers.StudentCourses;

[ApiController]
[Authorize]
[Route("studentcourses")]
public sealed class StudentCourseController(ISender sender) : ApiController
{
    [HttpGet(Name = "ListStudentCourses")]
    public async Task<ActionResult<PagedResult<StudentCourseDto>>> List([FromQuery] int? page, [FromQuery] int? pageSize, CancellationToken cancellationToken)
    {
        try
        {
            var result = await sender.Send(new ListStudentCourseQuery(page, pageSize), cancellationToken);
            return result.IsError
                ? Problem(result.Errors)
                : Ok(result.Value);
        }
        catch (ArgumentOutOfRangeException ex)
        {
            return BadRequest(new { error = ex.Message });
        }
    }

    [HttpGet("{id:guid}", Name = "GetStudentCourseById")]
    public async Task<ActionResult<StudentCourseDto>> Get(Guid id, CancellationToken cancellationToken)
    {
        var result = await sender.Send(new GetStudentCourseByIdQuery(id), cancellationToken);
        return result.IsError
            ? Problem(result.Errors)
            : Ok(result.Value);
    }

    [HttpPost(Name = "CreateStudentCourse")]
    public async Task<ActionResult<StudentCourseDto>> Create([FromBody] CreateStudentCourseRequest request, CancellationToken cancellationToken)
    {
        var created = await sender.Send(new CreateStudentCourseCommand
        {
            EnrolledAt = request.EnrolledAt,
            CourseId = request.CourseId,
            StudentId = request.StudentId,
        }, cancellationToken);
        if (created.IsError)
        {
            return Problem(created.Errors);
        }
        return CreatedAtRoute("GetStudentCourseById", new { id = created.Value.Id }, created.Value);
    }

    [HttpPut("{id:guid}", Name = "UpdateStudentCourse")]
    public async Task<ActionResult<StudentCourseDto>> Update(Guid id, [FromBody] UpdateStudentCourseRequest request, CancellationToken cancellationToken)
    {
        var updated = await sender.Send(new UpdateStudentCourseCommand
        {
            Id = id,
            EnrolledAt = request.EnrolledAt,
            CourseId = request.CourseId,
            StudentId = request.StudentId,
            ConcurrencyToken = request.ConcurrencyToken,
        }, cancellationToken);
        return updated.IsError
            ? Problem(updated.Errors)
            : Ok(updated.Value);
    }

    [HttpDelete("{id:guid}", Name = "DeleteStudentCourse")]
    public async Task<IActionResult> Delete(Guid id, [FromQuery] string? concurrencyToken, CancellationToken cancellationToken)
    {
        var deleted = await sender.Send(new DeleteStudentCourseCommand(id, concurrencyToken ?? string.Empty), cancellationToken);
        if (deleted.IsError)
        {
            return Problem(deleted.Errors);
        }
        return NoContent();
    }
}
