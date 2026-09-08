using ErrorOr;
using MediatR;
using SchoolService.Application.StudentCourses.Dtos;

namespace SchoolService.Application.StudentCourses.Commands.Create;

public sealed record CreateStudentCourseCommand : IRequest<ErrorOr<StudentCourseDto>>
{
    public DateTime EnrolledAt { get; init; }
    public Guid CourseId { get; init; }
    public Guid StudentId { get; init; }
}
