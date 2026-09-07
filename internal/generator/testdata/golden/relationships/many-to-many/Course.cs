namespace SchoolService.Domain.Entities;



public sealed class CourseState
{
    public required string Title { get; init; }
}

public sealed class Course
{
    private Course() { }

    public Guid Id { get; private set; }
    public string Title { get; private set; } = string.Empty;
    public ICollection<StudentCourse> StudentCourses { get; private set; } = [];

    public byte[] RowVersion { get; private set; } = [];

    public static Course Create(CourseState state) => new()
    {
        Id = Guid.NewGuid(),
        Title = state.Title,
    };

    public void Update(CourseState state)
    {
        Title = state.Title;
    }
}
