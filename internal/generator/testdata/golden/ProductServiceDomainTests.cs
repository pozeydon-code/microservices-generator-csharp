using ProductService.Domain.ValueObjects;

using Xunit;

namespace ProductService.UnitTests.Domain;

public sealed class ProductServiceDomainTests
{
    [Fact]
    public void ProductNameAcceptsValidValuesAndSupportsEqualityAndRehydration()
    {
        var first = ProductName.Create("Product Prime").Value!;
        var second = ProductName.Create("Product Prime").Value!;
        var rehydrated = ProductName.Rehydrate("Product Prime");

        Assert.Equal(first, second);
        Assert.Equal(first, rehydrated);
        Assert.Equal(first.GetHashCode(), second.GetHashCode());
        Assert.NotNull(first);
        Assert.Equal("Product Prime", first.Value);
    }

    [Fact]
    public void ProductNameUsesValueForUnequalEqualityAndHashSemantics()
    {
        var first = ProductName.Create("Product Prime").Value!;
        var different = ProductName.Create("Product Prime2").Value!;

        Assert.NotEqual(first, different);
        Assert.NotEqual(first.GetHashCode(), different.GetHashCode());
    }



    [Fact]
    public void ProductNameRejectsNull()
    {
        var result = ProductName.Create(null, "Field");

        Assert.True(result.IsError);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.Null", error.Code);
        Assert.Equal("Field", error.Metadata!["field"]);
    }

    [Fact]
    public void ProductNameAcceptsExactMinimumLengthBoundary()
    {
        var value = new string('x', 3);
        var result = ProductName.Create(value);
        Assert.False(result.IsError);
        Assert.Equal(value, result.Value!.Value);
    }

    [Fact]
    public void ProductNameAcceptsExactMaximumLengthBoundary()
    {
        var value = new string('x', 100);
        var result = ProductName.Create(value);
        Assert.False(result.IsError);
        Assert.Equal(value, result.Value!.Value);
    }


    [Fact]
    public void ProductNameAcceptsValidPatternExampleAndRejectsInvalidPatternExample()
    {
        Assert.False(ProductName.Create("Product Prime").IsError);
        Assert.Equal("ProductName.Pattern", Assert.Single(ProductName.Create("***").Errors).Code);
    }


    [Fact]
    public void ProductNameRejectsRequired()
    {
        var result = ProductName.Create("", "Field");

        Assert.True(result.IsError);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.Required", error.Code);
        Assert.Equal("Field", error.Metadata!["field"]);
        var exception = Assert.Throws<InvalidOperationException>(() => ProductName.Rehydrate(""));
        Assert.Equal("Failed to rehydrate ProductName: ProductName.Required", exception.Message);
    }

    [Fact]
    public void ProductNameRejectsMinLength()
    {
        var result = ProductName.Create("x", "Field");

        Assert.True(result.IsError);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.MinLength", error.Code);
        Assert.Equal("Field", error.Metadata!["field"]);
        var exception = Assert.Throws<InvalidOperationException>(() => ProductName.Rehydrate("x"));
        Assert.Equal("Failed to rehydrate ProductName: ProductName.MinLength", exception.Message);
    }

    [Fact]
    public void ProductNameRejectsMaxLength()
    {
        var result = ProductName.Create(new string('x', 100 + 1), "Field");

        Assert.True(result.IsError);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.MaxLength", error.Code);
        Assert.Equal("Field", error.Metadata!["field"]);
        var exception = Assert.Throws<InvalidOperationException>(() => ProductName.Rehydrate(new string('x', 100 + 1)));
        Assert.Equal("Failed to rehydrate ProductName: ProductName.MaxLength", exception.Message);
    }

    [Fact]
    public void ProductNameRejectsPattern()
    {
        var result = ProductName.Create("***", "Field");

        Assert.True(result.IsError);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.Pattern", error.Code);
        Assert.Equal("Field", error.Metadata!["field"]);
        var exception = Assert.Throws<InvalidOperationException>(() => ProductName.Rehydrate("***"));
        Assert.Equal("Failed to rehydrate ProductName: ProductName.Pattern", exception.Message);
    }

    [Fact]
    public void ProductPriceAcceptsValidValuesAndSupportsEqualityAndRehydration()
    {
        var first = ProductPrice.Create(0m).Value!;
        var second = ProductPrice.Create(0m).Value!;
        var rehydrated = ProductPrice.Rehydrate(0m);

        Assert.Equal(first, second);
        Assert.Equal(first, rehydrated);
        Assert.Equal(first.GetHashCode(), second.GetHashCode());
        Assert.NotNull(first);
        Assert.Equal(0m, first.Value);
    }

    [Fact]
    public void ProductPriceUsesValueForUnequalEqualityAndHashSemantics()
    {
        var first = ProductPrice.Create(0m).Value!;
        var different = ProductPrice.Create(999999.99m).Value!;

        Assert.NotEqual(first, different);
        Assert.NotEqual(first.GetHashCode(), different.GetHashCode());
    }



    [Fact]
    public void ProductPriceAcceptsMinimumBoundary()
    {
        var result = ProductPrice.Create(0m);
        Assert.False(result.IsError);
        Assert.Equal(0m, result.Value!.Value);
    }

    [Fact]
    public void ProductPriceAcceptsMaximumBoundary()
    {
        var result = ProductPrice.Create(999999.99m);
        Assert.False(result.IsError);
        Assert.Equal(999999.99m, result.Value!.Value);
    }


    [Fact]
    public void ProductPriceRejectsMinimum()
    {
        var result = ProductPrice.Create(0m - 1m, "Field");

        Assert.True(result.IsError);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductPrice.Minimum", error.Code);
        Assert.Equal("Field", error.Metadata!["field"]);
        var exception = Assert.Throws<InvalidOperationException>(() => ProductPrice.Rehydrate(0m - 1m));
        Assert.Equal("Failed to rehydrate ProductPrice: ProductPrice.Minimum", exception.Message);
    }

    [Fact]
    public void ProductPriceRejectsMaximum()
    {
        var result = ProductPrice.Create(999999.99m + 1m, "Field");

        Assert.True(result.IsError);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductPrice.Maximum", error.Code);
        Assert.Equal("Field", error.Metadata!["field"]);
        var exception = Assert.Throws<InvalidOperationException>(() => ProductPrice.Rehydrate(999999.99m + 1m));
        Assert.Equal("Failed to rehydrate ProductPrice: ProductPrice.Maximum", exception.Message);
    }

}
