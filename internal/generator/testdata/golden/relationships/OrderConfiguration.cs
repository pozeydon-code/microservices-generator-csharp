using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using OrderingService.Domain.Entities;


namespace OrderingService.Infrastructure.Persistence.Configurations;

public sealed class OrderConfiguration : IEntityTypeConfiguration<Order>
{
    public void Configure(EntityTypeBuilder<Order> builder)
    {
        builder.HasKey(item => item.Id);
        builder.Property(item => item.RowVersion).IsRowVersion();
        builder.Property(item => item.Number).IsRequired();

        builder.Property(item => item.CustomerId).IsRequired(false);


        builder.HasOne(item => item.Customer)
            .WithMany(item => item.Orders)
            .HasForeignKey(item => item.CustomerId)
            .IsRequired(false)
            .OnDelete(DeleteBehavior.Restrict);

    }
}
