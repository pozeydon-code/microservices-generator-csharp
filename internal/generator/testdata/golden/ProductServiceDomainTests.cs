using ProductService.Domain.Shared;
using ProductService.Domain.Shared.ValueObjects;

using Xunit;

namespace ProductService.Domain.Tests;

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

        Assert.False(result.IsSuccess);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.Null", error.Code);
        Assert.Equal("Field", error.Field);
    }

    [Fact]
    public void ProductNameAcceptsExactMinimumLengthBoundary()
    {
        var value = new string('x', 3);
        var result = ProductName.Create(value);
        Assert.True(result.IsSuccess);
        Assert.Equal(value, result.Value!.Value);
    }

    [Fact]
    public void ProductNameAcceptsExactMaximumLengthBoundary()
    {
        var value = new string('x', 100);
        var result = ProductName.Create(value);
        Assert.True(result.IsSuccess);
        Assert.Equal(value, result.Value!.Value);
    }


    [Fact]
    public void ProductNameAcceptsValidPatternExampleAndRejectsInvalidPatternExample()
    {
        Assert.True(ProductName.Create("Product Prime").IsSuccess);
        Assert.Equal("ProductName.Pattern", Assert.Single(ProductName.Create("***").Errors).Code);
    }


    [Fact]
    public void ProductNameRejectsRequired()
    {
        var result = ProductName.Create("", "Field");

        Assert.False(result.IsSuccess);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.Required", error.Code);
        Assert.Equal("Field", error.Field);
        var exception = Assert.Throws<DomainReconstitutionException>(() => ProductName.Rehydrate(""));
        Assert.Equal("ProductName", exception.ValueObjectType);
        Assert.Contains("ProductName.Required", exception.InvariantCodes);
    }

    [Fact]
    public void ProductNameRejectsMinLength()
    {
        var result = ProductName.Create("x", "Field");

        Assert.False(result.IsSuccess);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.MinLength", error.Code);
        Assert.Equal("Field", error.Field);
        var exception = Assert.Throws<DomainReconstitutionException>(() => ProductName.Rehydrate("x"));
        Assert.Equal("ProductName", exception.ValueObjectType);
        Assert.Contains("ProductName.MinLength", exception.InvariantCodes);
    }

    [Fact]
    public void ProductNameRejectsMaxLength()
    {
        var result = ProductName.Create(new string('x', 100 + 1), "Field");

        Assert.False(result.IsSuccess);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.MaxLength", error.Code);
        Assert.Equal("Field", error.Field);
        var exception = Assert.Throws<DomainReconstitutionException>(() => ProductName.Rehydrate(new string('x', 100 + 1)));
        Assert.Equal("ProductName", exception.ValueObjectType);
        Assert.Contains("ProductName.MaxLength", exception.InvariantCodes);
    }

    [Fact]
    public void ProductNameRejectsPattern()
    {
        var result = ProductName.Create("***", "Field");

        Assert.False(result.IsSuccess);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductName.Pattern", error.Code);
        Assert.Equal("Field", error.Field);
        var exception = Assert.Throws<DomainReconstitutionException>(() => ProductName.Rehydrate("***"));
        Assert.Equal("ProductName", exception.ValueObjectType);
        Assert.Contains("ProductName.Pattern", exception.InvariantCodes);
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
        Assert.True(result.IsSuccess);
        Assert.Equal(0m, result.Value!.Value);
    }

    [Fact]
    public void ProductPriceAcceptsMaximumBoundary()
    {
        var result = ProductPrice.Create(999999.99m);
        Assert.True(result.IsSuccess);
        Assert.Equal(999999.99m, result.Value!.Value);
    }


    [Fact]
    public void ProductPriceRejectsMinimum()
    {
        var result = ProductPrice.Create(0m - 1m, "Field");

        Assert.False(result.IsSuccess);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductPrice.Minimum", error.Code);
        Assert.Equal("Field", error.Field);
        var exception = Assert.Throws<DomainReconstitutionException>(() => ProductPrice.Rehydrate(0m - 1m));
        Assert.Equal("ProductPrice", exception.ValueObjectType);
        Assert.Contains("ProductPrice.Minimum", exception.InvariantCodes);
    }

    [Fact]
    public void ProductPriceRejectsMaximum()
    {
        var result = ProductPrice.Create(999999.99m + 1m, "Field");

        Assert.False(result.IsSuccess);
        var error = Assert.Single(result.Errors);
        Assert.Equal("ProductPrice.Maximum", error.Code);
        Assert.Equal("Field", error.Field);
        var exception = Assert.Throws<DomainReconstitutionException>(() => ProductPrice.Rehydrate(999999.99m + 1m));
        Assert.Equal("ProductPrice", exception.ValueObjectType);
        Assert.Contains("ProductPrice.Maximum", exception.InvariantCodes);
    }

}
