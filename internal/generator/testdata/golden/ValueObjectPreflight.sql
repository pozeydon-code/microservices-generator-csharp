-- Generated Value Object preflight for ProductService.
-- Run before rule-tightening migrations and after repairs.
-- Regex rules are intentionally not translated to SQL; audit those with application/domain validation because Go/.NET/SQL regex semantics are not safely portable.
SET NOCOUNT ON;

DECLARE @Violations TABLE (
    TableName sysname NOT NULL,
    ColumnName sysname NOT NULL,
    RuleCode nvarchar(256) NOT NULL,
    RecordId uniqueidentifier NOT NULL
);

INSERT INTO @Violations SELECT N'Products', N'Name', N'ProductName.MaxLength', [Id] FROM [dbo].[Products] WHERE [Name] IS NOT NULL AND LEN([Name]) > 100;
INSERT INTO @Violations SELECT N'Products', N'Name', N'ProductName.Required', [Id] FROM [dbo].[Products] WHERE [Name] IS NULL OR LTRIM(RTRIM([Name])) = N'';

INSERT INTO @Violations SELECT N'Products', N'Price', N'ProductPrice.Minimum', [Id] FROM [dbo].[Products] WHERE [Price] < 0;
INSERT INTO @Violations SELECT N'Products', N'Price', N'ProductPrice.Maximum', [Id] FROM [dbo].[Products] WHERE [Price] > 999999.99;

SELECT TableName, ColumnName, RuleCode, RecordId
FROM @Violations
ORDER BY TableName, ColumnName, RuleCode, RecordId;
