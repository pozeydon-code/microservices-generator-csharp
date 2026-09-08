namespace SchoolService.Application.StudentCourses.Dtos;

public sealed record StudentCourseDto(
    DateTime EnrolledAt,
    Guid Id,
    Guid CourseId,
    Guid StudentId,
    string ConcurrencyToken);
