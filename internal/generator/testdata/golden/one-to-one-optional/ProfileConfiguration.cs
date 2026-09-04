using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using IdentityService.Domain.Entities;


namespace IdentityService.Infrastructure.Persistence.Configurations;

public sealed class ProfileConfiguration : IEntityTypeConfiguration<Profile>
{
    public void Configure(EntityTypeBuilder<Profile> builder)
    {
        builder.HasKey(item => item.Id);
        builder.Property(item => item.RowVersion).IsRowVersion();
        builder.Property(item => item.DisplayName).IsRequired();

        builder.Property(item => item.UserId).IsRequired(false);


        builder.HasOne(item => item.User)
            .WithOne(item => item.Profile)
            .HasForeignKey<Profile>(item => item.UserId)
            .IsRequired(false)
            .OnDelete(DeleteBehavior.Restrict);

    }
}
