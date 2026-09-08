namespace SchoolService.Domain.Entities;



public sealed class StudentCourseState
{
    public required DateTime EnrolledAt { get; init; }
    public required Guid CourseId { get; init; }
    public required Guid StudentId { get; init; }
}

public sealed class StudentCourse
{
    private StudentCourse() { }

    public DateTime EnrolledAt { get; private set; }
    public Guid Id { get; private set; }
    public Guid CourseId { get; private set; }
    public Guid StudentId { get; private set; }
    public Course Course { get; private set; } = null!;
    public Student Student { get; private set; } = null!;

    public byte[] RowVersion { get; private set; } = [];

    public static StudentCourse Create(StudentCourseState state) => new()
    {
        Id = Guid.NewGuid(),
        EnrolledAt = state.EnrolledAt,
        CourseId = state.CourseId,
        StudentId = state.StudentId,
    };

    public void Update(StudentCourseState state)
    {
        EnrolledAt = state.EnrolledAt;
        CourseId = state.CourseId;
        StudentId = state.StudentId;
    }
}
