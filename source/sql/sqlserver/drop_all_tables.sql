-- https://stackoverflow.com/a/8439798/9731532
DECLARE @Sql NVARCHAR(500) DECLARE @Cursor CURSOR

-- using QUOTENAME gives me an error for some reason.
SET @Cursor = CURSOR FAST_FORWARD FOR
SELECT DISTINCT sql = 'ALTER TABLE [' + tc2.TABLE_SCHEMA + '].[' +  tc2.TABLE_NAME + '] DROP [' + rc1.CONSTRAINT_NAME + '];'
FROM INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc1
LEFT JOIN INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc2 ON tc2.CONSTRAINT_NAME = rc1.CONSTRAINT_NAME

OPEN @Cursor FETCH NEXT FROM @Cursor INTO @Sql

WHILE (@@FETCH_STATUS = 0)
BEGIN
Exec sp_executesql @Sql
FETCH NEXT FROM @Cursor INTO @Sql
END

CLOSE @Cursor DEALLOCATE @Cursor
GO

EXEC sp_MSforeachtable 'DROP TABLE ?'
GO


-- Drop all partition schemes and functions
DECLARE @schemeName NVARCHAR(128);

DECLARE scheme_cursor CURSOR FOR
SELECT name
FROM sys.partition_schemes;

OPEN scheme_cursor;
FETCH NEXT FROM scheme_cursor INTO @schemeName;

WHILE @@FETCH_STATUS = 0
BEGIN
    EXEC ('DROP PARTITION SCHEME [' + @schemeName + '];');
    FETCH NEXT FROM scheme_cursor INTO @schemeName;
END

CLOSE scheme_cursor;
DEALLOCATE scheme_cursor;

-- Drop all partition functions using a cursor
DECLARE @functionName NVARCHAR(128);

DECLARE function_cursor CURSOR FOR
SELECT name
FROM sys.partition_functions;

OPEN function_cursor;
FETCH NEXT FROM function_cursor INTO @functionName;

WHILE @@FETCH_STATUS = 0
BEGIN
    EXEC ('DROP PARTITION FUNCTION [' + @functionName + '];');
    FETCH NEXT FROM function_cursor INTO @functionName;
END

CLOSE function_cursor;
DEALLOCATE function_cursor;
