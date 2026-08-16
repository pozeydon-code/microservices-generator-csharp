using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using ProductService.Domain.Entities;
using ProductService.Domain.ValueObjects;


namespace ProductService.Infrastructure.Persistence.Configurations;

public sealed class ProductConfiguration : IEntityTypeConfiguration<Product>
{
    public void Configure(EntityTypeBuilder<Product> builder)
    {
        builder.HasKey(item => item.Id);
        builder.Property<byte[]>("RowVersion").IsRowVersion();
        builder.Property(item => item.IsActive).IsRequired();

        builder.Property(item => item.Name)
            .HasConversion(value => value.Value, value => ProductService.Domain.ValueObjects.ProductName.Rehydrate(value))
            .HasMaxLength(100)
            .IsRequired();
        builder.Property(item => item.Price)
            .HasConversion(value => value.Value, value => ProductService.Domain.ValueObjects.ProductPrice.Rehydrate(value))
            .IsRequired();

    }
}
