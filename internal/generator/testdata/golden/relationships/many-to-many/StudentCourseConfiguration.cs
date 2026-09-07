using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using SchoolService.Domain.Entities;


namespace SchoolService.Infrastructure.Persistence.Configurations;

public sealed class StudentCourseConfiguration : IEntityTypeConfiguration<StudentCourse>
{
    public void Configure(EntityTypeBuilder<StudentCourse> builder)
    {
        builder.HasKey(item => item.Id);
        builder.Property(item => item.RowVersion).IsRowVersion();
        builder.Property(item => item.EnrolledAt).IsRequired();

        builder.Property(item => item.CourseId).IsRequired();

        builder.Property(item => item.StudentId).IsRequired();


        builder.HasOne(item => item.Course)
            .WithMany(item => item.StudentCourses)
            .HasForeignKey(item => item.CourseId)
            .IsRequired()
            .OnDelete(DeleteBehavior.Restrict);

        builder.HasOne(item => item.Student)
            .WithMany(item => item.StudentCourses)
            .HasForeignKey(item => item.StudentId)
            .IsRequired()
            .OnDelete(DeleteBehavior.Restrict);

    }
}
