using Microsoft.Data.SqlClient;
using Microsoft.EntityFrameworkCore;
using Microsoft.EntityFrameworkCore.Design;

namespace ProductService.Infrastructure.Persistence;

public sealed class ProductServiceDbContextFactory : IDesignTimeDbContextFactory<ProductServiceDbContext>
{
    private const string DesignTimeConnectionString = "Data Source=(localdb)\\MSSQLLocalDB;Initial Catalog=ProductService;Integrated Security=True;Encrypt=False;TrustServerCertificate=True";

    public ProductServiceDbContext CreateDbContext(string[] args)
    {
        var connectionString = Environment.GetEnvironmentVariable("ConnectionStrings__DefaultConnection");
        if (string.IsNullOrWhiteSpace(connectionString))
        {
            connectionString = DesignTimeConnectionString;
        }

        var connectionBuilder = new SqlConnectionStringBuilder(connectionString)
        {
            ConnectTimeout = Math.Min(Math.Max(new SqlConnectionStringBuilder(connectionString).ConnectTimeout, 1), ResiliencePolicy.SqlConnectionTimeoutSeconds)
        };

        var options = new DbContextOptionsBuilder<ProductServiceDbContext>()
            .UseSqlServer(
                connectionBuilder.ConnectionString,
                sql => sql.EnableRetryOnFailure(maxRetryCount: ResiliencePolicy.SqlRetryCount, maxRetryDelay: ResiliencePolicy.SqlRetryDelay, errorNumbersToAdd: null).CommandTimeout(ResiliencePolicy.SqlCommandTimeoutSeconds))
            .Options;

        return new ProductServiceDbContext(options);
    }
}
