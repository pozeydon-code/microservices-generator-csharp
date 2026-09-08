namespace SchoolService.Application.StudentCourses.Dtos;

public sealed record CreateStudentCourseRequest
{
    public DateTime EnrolledAt { get; init; }
    public Guid CourseId { get; init; }
    public Guid StudentId { get; init; }
}
