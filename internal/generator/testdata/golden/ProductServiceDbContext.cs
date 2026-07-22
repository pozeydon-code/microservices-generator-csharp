using Microsoft.EntityFrameworkCore;
using ProductService.Domain.Features.Products;
using ProductService.Domain.Common.ValueObjects;


namespace ProductService.Infrastructure.Persistence;

public sealed class ProductServiceDbContext(DbContextOptions<ProductServiceDbContext> options) : DbContext(options)
{
    public DbSet<Product> Products => Set<Product>();

    protected override void OnModelCreating(ModelBuilder modelBuilder)
    {
        modelBuilder.Entity<Product>(entity =>
        {
            entity.HasKey(item => item.Id);
            entity.Property<byte[]>("RowVersion").IsRowVersion();
            entity.Property(item => item.IsActive).IsRequired();

            entity.Property(item => item.Name)
                .HasConversion(value => value.Value, value => ProductName.Rehydrate(value))
                .HasMaxLength(100)
                .IsRequired();
            entity.Property(item => item.Price)
                .HasConversion(value => value.Value, value => ProductPrice.Rehydrate(value))
                .IsRequired();

        });
    }
}
