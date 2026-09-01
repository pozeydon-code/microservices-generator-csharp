using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Metadata.Builders;
using OrderingService.Domain.Entities;


namespace OrderingService.Infrastructure.Persistence.Configurations;

public sealed class OrderItemConfiguration : IEntityTypeConfiguration<OrderItem>
{
    public void Configure(EntityTypeBuilder<OrderItem> builder)
    {
        builder.HasKey(item => item.Id);
        builder.Property(item => item.RowVersion).IsRowVersion();
        builder.Property(item => item.Sku).IsRequired();

        builder.Property(item => item.OrderId).IsRequired();


        builder.HasOne(item => item.Order)
            .WithMany(item => item.Items)
            .HasForeignKey(item => item.OrderId)
            .IsRequired()
            .OnDelete(DeleteBehavior.Restrict);

    }
}
