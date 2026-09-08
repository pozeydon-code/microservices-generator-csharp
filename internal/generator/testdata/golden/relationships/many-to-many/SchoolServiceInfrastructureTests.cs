using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;
using Microsoft.Extensions.Logging.Abstractions;
using SchoolService.Infrastructure;
using SchoolService.Infrastructure.Health;
using SchoolService.Application.Common;
using SchoolService.Domain.Entities;
using SchoolService.Application.Courses.Interfaces;
using SchoolService.Infrastructure.Persistence.Features.Courses;
using DomainCourse = SchoolService.Domain.Entities.Course;
using SchoolService.Application.Students.Interfaces;
using SchoolService.Infrastructure.Persistence.Features.Students;
using DomainStudent = SchoolService.Domain.Entities.Student;
using SchoolService.Application.StudentCourses.Interfaces;
using SchoolService.Infrastructure.Persistence.Features.StudentCourses;
using DomainStudentCourse = SchoolService.Domain.Entities.StudentCourse;

using SchoolService.Infrastructure.Persistence;
using Xunit;

namespace SchoolService.Infrastructure.Tests;

public sealed class SchoolServiceInfrastructureTests
{
    [Fact]
    public async Task ReadinessValidatesDatabaseConnectivityAndMappedColumns()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"SchoolService_readiness_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);

        await using (var beforeContext = database.CreateContext())
        {
            var before = await new SqlReadinessProbe(beforeContext, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
            Assert.Equal(ReadinessStatus.NotReady, before.Status);
        }

        await database.InitializeAsync();
        await using (var healthyContext = database.CreateContext())
        {
            var healthy = await new SqlReadinessProbe(healthyContext, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
            Assert.Equal(ReadinessStatus.Ready, healthy.Status);
        }

        await using (var schemaChangedContext = database.CreateContext())
        {
            await schemaChangedContext.Database.ExecuteSqlRawAsync("ALTER TABLE Courses DROP COLUMN Title");
            var schemaChanged = await new SqlReadinessProbe(schemaChangedContext, NullLogger<SqlReadinessProbe>.Instance).CheckAsync(CancellationToken.None);
            Assert.Equal(ReadinessStatus.NotReady, schemaChanged.Status);
        }
    }

    [Fact]
    public async Task CourseRepositoryPersistsReloadsConcurrencyAndConflicts()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"SchoolService_Course_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var createContext = database.CreateContext();
        var repository = new CourseRepository(createContext, NullLogger<CourseRepository>.Instance);
        var unitOfWork = new UnitOfWork(createContext);
        var entity = DomainCourse.Create(new CourseState { Title = "Title Value",  });
        await repository.AddAsync(entity, CancellationToken.None);
        await unitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await repository.GetByIdAsync(entity.Id, CancellationToken.None);
        Assert.NotNull(created);
        Assert.False(string.IsNullOrWhiteSpace(created.ConcurrencyToken));
        Assert.True(EqualityComparer<string>.Default.Equals("Title Value", created.Entity.Title));

        await using var reloadContext = database.CreateContext();
        var reloadRepository = new CourseRepository(reloadContext, NullLogger<CourseRepository>.Instance);
        var reloadUnitOfWork = new UnitOfWork(reloadContext);
        var reloaded = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(reloaded);
        Assert.True(EqualityComparer<string>.Default.Equals("Title Value", reloaded.Entity.Title));

        Assert.Equal(created.ConcurrencyToken, reloaded.ConcurrencyToken);

        reloaded.Entity.Update(new CourseState { Title = "Updated Title",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(reloaded.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var updated = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(updated);
        Assert.NotEqual(reloaded.ConcurrencyToken, updated.ConcurrencyToken);
        Assert.True(EqualityComparer<string>.Default.Equals("Updated Title", updated.Entity.Title));

        updated.Entity.Update(new CourseState { Title = "Title Value",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));

        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, updated.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        Assert.Null(await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None));
    }

    [Fact]
    public async Task CourseRepositoryMapsTwoContextConcurrencyRaceToConflict()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"SchoolService_Course_race_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var seedContext = database.CreateContext();
        var seedRepository = new CourseRepository(seedContext, NullLogger<CourseRepository>.Instance);
        var seedUnitOfWork = new UnitOfWork(seedContext);
        var seed = DomainCourse.Create(new CourseState { Title = "Title Value",  });
        await seedRepository.AddAsync(seed, CancellationToken.None);
        await seedUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await seedRepository.GetByIdAsync(seed.Id, CancellationToken.None);
        Assert.NotNull(created);

        await using var firstContext = database.CreateContext();
        await using var secondContext = database.CreateContext();
        var firstRepository = new CourseRepository(firstContext, NullLogger<CourseRepository>.Instance);
        var secondRepository = new CourseRepository(secondContext, NullLogger<CourseRepository>.Instance);
        var firstUnitOfWork = new UnitOfWork(firstContext);
        var secondUnitOfWork = new UnitOfWork(secondContext);
        var first = await firstRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        var second = await secondRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(first);
        Assert.NotNull(second);

        second.Entity.Update(new CourseState { Title = "Updated Title",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await secondRepository.UpdateAsync(second.Entity, second.ConcurrencyToken, CancellationToken.None));
        await secondUnitOfWork.SaveChangesAsync(CancellationToken.None);
        first.Entity.Update(new CourseState { Title = "Title Value",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await firstRepository.UpdateAsync(first.Entity, first.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => firstUnitOfWork.SaveChangesAsync(CancellationToken.None));
    }


    [Fact]
    public async Task StudentRepositoryPersistsReloadsConcurrencyAndConflicts()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"SchoolService_Student_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var createContext = database.CreateContext();
        var repository = new StudentRepository(createContext, NullLogger<StudentRepository>.Instance);
        var unitOfWork = new UnitOfWork(createContext);
        var entity = DomainStudent.Create(new StudentState { Name = "Name Value",  });
        await repository.AddAsync(entity, CancellationToken.None);
        await unitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await repository.GetByIdAsync(entity.Id, CancellationToken.None);
        Assert.NotNull(created);
        Assert.False(string.IsNullOrWhiteSpace(created.ConcurrencyToken));
        Assert.True(EqualityComparer<string>.Default.Equals("Name Value", created.Entity.Name));

        await using var reloadContext = database.CreateContext();
        var reloadRepository = new StudentRepository(reloadContext, NullLogger<StudentRepository>.Instance);
        var reloadUnitOfWork = new UnitOfWork(reloadContext);
        var reloaded = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(reloaded);
        Assert.True(EqualityComparer<string>.Default.Equals("Name Value", reloaded.Entity.Name));

        Assert.Equal(created.ConcurrencyToken, reloaded.ConcurrencyToken);

        reloaded.Entity.Update(new StudentState { Name = "Updated Name",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(reloaded.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var updated = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(updated);
        Assert.NotEqual(reloaded.ConcurrencyToken, updated.ConcurrencyToken);
        Assert.True(EqualityComparer<string>.Default.Equals("Updated Name", updated.Entity.Name));

        updated.Entity.Update(new StudentState { Name = "Name Value",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));

        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, updated.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        Assert.Null(await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None));
    }

    [Fact]
    public async Task StudentRepositoryMapsTwoContextConcurrencyRaceToConflict()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"SchoolService_Student_race_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var seedContext = database.CreateContext();
        var seedRepository = new StudentRepository(seedContext, NullLogger<StudentRepository>.Instance);
        var seedUnitOfWork = new UnitOfWork(seedContext);
        var seed = DomainStudent.Create(new StudentState { Name = "Name Value",  });
        await seedRepository.AddAsync(seed, CancellationToken.None);
        await seedUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await seedRepository.GetByIdAsync(seed.Id, CancellationToken.None);
        Assert.NotNull(created);

        await using var firstContext = database.CreateContext();
        await using var secondContext = database.CreateContext();
        var firstRepository = new StudentRepository(firstContext, NullLogger<StudentRepository>.Instance);
        var secondRepository = new StudentRepository(secondContext, NullLogger<StudentRepository>.Instance);
        var firstUnitOfWork = new UnitOfWork(firstContext);
        var secondUnitOfWork = new UnitOfWork(secondContext);
        var first = await firstRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        var second = await secondRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(first);
        Assert.NotNull(second);

        second.Entity.Update(new StudentState { Name = "Updated Name",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await secondRepository.UpdateAsync(second.Entity, second.ConcurrencyToken, CancellationToken.None));
        await secondUnitOfWork.SaveChangesAsync(CancellationToken.None);
        first.Entity.Update(new StudentState { Name = "Name Value",  });
        Assert.Equal(MutationPreparationStatus.Prepared, await firstRepository.UpdateAsync(first.Entity, first.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => firstUnitOfWork.SaveChangesAsync(CancellationToken.None));
    }


    [Fact]
    public async Task StudentCourseRepositoryPersistsReloadsConcurrencyAndConflicts()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"SchoolService_StudentCourse_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var createContext = database.CreateContext();
        await SeedRequiredRelationshipsForStudentCourseAsync(createContext);

        var repository = new StudentCourseRepository(createContext, NullLogger<StudentCourseRepository>.Instance);
        var unitOfWork = new UnitOfWork(createContext);
        var entity = DomainStudentCourse.Create(new StudentCourseState { EnrolledAt = new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc), CourseId = Guid.Parse("00000000-0000-0000-0000-000000000001"), StudentId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        await repository.AddAsync(entity, CancellationToken.None);
        await unitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await repository.GetByIdAsync(entity.Id, CancellationToken.None);
        Assert.NotNull(created);
        Assert.False(string.IsNullOrWhiteSpace(created.ConcurrencyToken));
        Assert.True(EqualityComparer<DateTime>.Default.Equals(new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc), created.Entity.EnrolledAt));

        await using var reloadContext = database.CreateContext();
        var reloadRepository = new StudentCourseRepository(reloadContext, NullLogger<StudentCourseRepository>.Instance);
        var reloadUnitOfWork = new UnitOfWork(reloadContext);
        var reloaded = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(reloaded);
        Assert.True(EqualityComparer<DateTime>.Default.Equals(new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc), reloaded.Entity.EnrolledAt));

        Assert.Equal(created.ConcurrencyToken, reloaded.ConcurrencyToken);

        reloaded.Entity.Update(new StudentCourseState { EnrolledAt = new DateTime(2024, 2, 1, 0, 0, 0, DateTimeKind.Utc), CourseId = Guid.Parse("00000000-0000-0000-0000-000000000002"), StudentId = Guid.Parse("00000000-0000-0000-0000-000000000002"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(reloaded.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var updated = await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(updated);
        Assert.NotEqual(reloaded.ConcurrencyToken, updated.ConcurrencyToken);
        Assert.True(EqualityComparer<DateTime>.Default.Equals(new DateTime(2024, 2, 1, 0, 0, 0, DateTimeKind.Utc), updated.Entity.EnrolledAt));

        updated.Entity.Update(new StudentCourseState { EnrolledAt = new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc), CourseId = Guid.Parse("00000000-0000-0000-0000-000000000001"), StudentId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.UpdateAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));

        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, reloaded.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, "not-base64", CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String([1]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[7]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[8]), CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => reloadUnitOfWork.SaveChangesAsync(CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.InvalidToken, await reloadRepository.DeleteAsync(updated.Entity, Convert.ToBase64String(new byte[9]), CancellationToken.None));
        Assert.Equal(MutationPreparationStatus.Prepared, await reloadRepository.DeleteAsync(updated.Entity, updated.ConcurrencyToken, CancellationToken.None));
        await reloadUnitOfWork.SaveChangesAsync(CancellationToken.None);
        Assert.Null(await reloadRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None));
    }

    [Fact]
    public async Task StudentCourseRepositoryMapsTwoContextConcurrencyRaceToConflict()
    {
        if (!SqlTestDatabase.IsConfigured) { return; }
        var databaseName = $"SchoolService_StudentCourse_race_{Guid.NewGuid():N}";
        await using var database = new SqlTestDatabase(databaseName);
        await database.InitializeAsync();

        await using var seedContext = database.CreateContext();
        await SeedRequiredRelationshipsForStudentCourseAsync(seedContext);

        var seedRepository = new StudentCourseRepository(seedContext, NullLogger<StudentCourseRepository>.Instance);
        var seedUnitOfWork = new UnitOfWork(seedContext);
        var seed = DomainStudentCourse.Create(new StudentCourseState { EnrolledAt = new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc), CourseId = Guid.Parse("00000000-0000-0000-0000-000000000001"), StudentId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        await seedRepository.AddAsync(seed, CancellationToken.None);
        await seedUnitOfWork.SaveChangesAsync(CancellationToken.None);
        var created = await seedRepository.GetByIdAsync(seed.Id, CancellationToken.None);
        Assert.NotNull(created);

        await using var firstContext = database.CreateContext();
        await using var secondContext = database.CreateContext();
        var firstRepository = new StudentCourseRepository(firstContext, NullLogger<StudentCourseRepository>.Instance);
        var secondRepository = new StudentCourseRepository(secondContext, NullLogger<StudentCourseRepository>.Instance);
        var firstUnitOfWork = new UnitOfWork(firstContext);
        var secondUnitOfWork = new UnitOfWork(secondContext);
        var first = await firstRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        var second = await secondRepository.GetByIdAsync(created.Entity.Id, CancellationToken.None);
        Assert.NotNull(first);
        Assert.NotNull(second);

        second.Entity.Update(new StudentCourseState { EnrolledAt = new DateTime(2024, 2, 1, 0, 0, 0, DateTimeKind.Utc), CourseId = Guid.Parse("00000000-0000-0000-0000-000000000002"), StudentId = Guid.Parse("00000000-0000-0000-0000-000000000002"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await secondRepository.UpdateAsync(second.Entity, second.ConcurrencyToken, CancellationToken.None));
        await secondUnitOfWork.SaveChangesAsync(CancellationToken.None);
        first.Entity.Update(new StudentCourseState { EnrolledAt = new DateTime(2024, 1, 1, 0, 0, 0, DateTimeKind.Utc), CourseId = Guid.Parse("00000000-0000-0000-0000-000000000001"), StudentId = Guid.Parse("00000000-0000-0000-0000-000000000001"),  });
        Assert.Equal(MutationPreparationStatus.Prepared, await firstRepository.UpdateAsync(first.Entity, first.ConcurrencyToken, CancellationToken.None));
        await Assert.ThrowsAsync<ConcurrencyConflictException>(() => firstUnitOfWork.SaveChangesAsync(CancellationToken.None));
    }



    private static async Task SeedRequiredRelationshipsForStudentCourseAsync(SchoolServiceDbContext context)
    {
        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Courses] ([Id], [Title]) VALUES ({0}, N'Title Value')", Guid.Parse("00000000-0000-0000-0000-000000000001"));
        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Courses] ([Id], [Title]) VALUES ({0}, N'Title Value')", Guid.Parse("00000000-0000-0000-0000-000000000002"));
        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Students] ([Id], [Name]) VALUES ({0}, N'Name Value')", Guid.Parse("00000000-0000-0000-0000-000000000001"));
        await context.Database.ExecuteSqlRawAsync("INSERT INTO [Students] ([Id], [Name]) VALUES ({0}, N'Name Value')", Guid.Parse("00000000-0000-0000-0000-000000000002"));
        await context.SaveChangesAsync(CancellationToken.None);
    }


    private sealed class CaptureLogger<T> : Microsoft.Extensions.Logging.ILogger<T>
    {
        public string LastMessage { get; private set; } = string.Empty;
        public IDisposable? BeginScope<TState>(TState state) where TState : notnull => null;
        public bool IsEnabled(Microsoft.Extensions.Logging.LogLevel logLevel) => true;
        public void Log<TState>(Microsoft.Extensions.Logging.LogLevel logLevel, Microsoft.Extensions.Logging.EventId eventId, TState state, Exception? exception, Func<TState, Exception?, string> formatter) => LastMessage = formatter(state, exception);
    }

    private sealed class SqlTestDatabase(string databaseName) : IAsyncDisposable
    {
        public static bool IsConfigured => !string.IsNullOrWhiteSpace(Environment.GetEnvironmentVariable("MICROGEN_TEST_SQLSERVER"));

        private readonly string connectionString = BuildConnectionString(databaseName);

        public async Task InitializeAsync()
        {
            await using var context = CreateContext();
            await context.Database.EnsureDeletedAsync();
            await context.Database.EnsureCreatedAsync();
        }

        public SchoolServiceDbContext CreateContext()
        {
            var options = new DbContextOptionsBuilder<SchoolServiceDbContext>()
                .UseSqlServer(connectionString, sql => sql.CommandTimeout(5))
                .Options;
            return new SchoolServiceDbContext(options);
        }

        public async ValueTask DisposeAsync()
        {
            await using var context = CreateContext();
            await context.Database.EnsureDeletedAsync();
        }

        private static string BuildConnectionString(string databaseName)
        {
            var server = Environment.GetEnvironmentVariable("MICROGEN_TEST_SQLSERVER");
            if (string.IsNullOrWhiteSpace(server))
            {
                throw new InvalidOperationException("Set MICROGEN_TEST_SQLSERVER to run generated Infrastructure SQL tests.");
            }
            var builder = new SqlConnectionStringBuilder(server) { InitialCatalog = databaseName, ConnectTimeout = 5, TrustServerCertificate = true };
            return builder.ConnectionString;
        }
    }
}
