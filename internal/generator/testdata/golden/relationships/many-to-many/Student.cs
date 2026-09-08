namespace SchoolService.Domain.Entities;



public sealed class StudentState
{
    public required string Name { get; init; }
}

public sealed class Student
{
    private Student() { }

    public Guid Id { get; private set; }
    public string Name { get; private set; } = string.Empty;
    public ICollection<StudentCourse> StudentCourses { get; private set; } = [];

    public byte[] RowVersion { get; private set; } = [];

    public static Student Create(StudentState state) => new()
    {
        Id = Guid.NewGuid(),
        Name = state.Name,
    };

    public void Update(StudentState state)
    {
        Name = state.Name;
    }
}
