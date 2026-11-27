using Microsoft.EntityFrameworkCore;
using ProductService.Application.Common;

namespace ProductService.Infrastructure.Persistence;

public sealed class UnitOfWork(ProductServiceDbContext dbContext) : IUnitOfWork
{
    public async Task<int> SaveChangesAsync(CancellationToken cancellationToken)
    {
        try
        {
            return await dbContext.SaveChangesAsync(cancellationToken);
        }
        catch (DbUpdateConcurrencyException)
        {
            throw new ConcurrencyConflictException();
        }
    }
}
