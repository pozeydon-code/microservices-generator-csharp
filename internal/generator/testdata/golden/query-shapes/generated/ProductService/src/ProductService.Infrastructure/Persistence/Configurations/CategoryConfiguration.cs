using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using ProductService.Domain.Entities;


namespace ProductService.Infrastructure.Persistence.Configurations;

public sealed class CategoryConfiguration : IEntityTypeConfiguration<Category>
{
    public void Configure(EntityTypeBuilder<Category> builder)
    {
        builder.HasKey(item => item.Id);
        builder.Property(item => item.RowVersion).IsRowVersion();
        builder.Property(item => item.Name).IsRequired();

        builder.Property(item => item.CategoryId).IsRequired();


        builder.HasOne(item => item.Product)
            .WithMany(item => item.Categorys)
            .HasForeignKey(item => item.CategoryId)
            .IsRequired()
            .OnDelete(DeleteBehavior.Restrict);

    }
}
