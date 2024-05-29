package main

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
)

type StringsByDatabase struct {
    Postgres string
    SqlServer string
}

func (strings *StringsByDatabase) Get(databaseType DatabaseType) string {
    switch (databaseType) {
    case Postgres:
        return strings.Postgres
    case SqlServer:
        return strings.SqlServer
    default:
        panic("unreachable")
    }
}

type helperStringsForType struct {
    allFields string
    allFieldsPrefixedBySource string
    allFieldsButIdSetToSourceField string
    allFieldsButIdSetToExcludedField string
    columnNames []string
}

func pascalToCamel(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func makeAssignList(names []string, prefix string) string {
    var r strings.Builder
    for i, name := range names {
        if (i != 0) {
            r.WriteString(", ")
        }
        r.WriteString(name)
        r.WriteString(" = ")
        r.WriteString(prefix)
        r.WriteString(".")
        r.WriteString(name)
    }
    return r.String()
}

func createHelperForType(t reflect.Type) helperStringsForType {
    // Get all fields, make first letter lowercase
    var columnNames []string
    for i := 0; i < t.NumField(); i++ {
        name := t.Field(i).Name
        camelName := pascalToCamel(name)
        columnNames = append(columnNames, camelName)
    }

    if columnNames[0] != "id" {
        panic("The first fields should be id")
    }

    allFields := strings.Join(columnNames, ", ")
    allFieldsPrefixedBySource := "source." + strings.Join(columnNames, ", source.")
    allFieldsButIdSetToSourceField := makeAssignList(columnNames[1:], "source")
    allFieldsButIdSetToExcludedField := makeAssignList(columnNames[1:], "excluded")
    return helperStringsForType{
        allFields: allFields,
        allFieldsPrefixedBySource: allFieldsPrefixedBySource,
        allFieldsButIdSetToSourceField: allFieldsButIdSetToSourceField,
        allFieldsButIdSetToExcludedField: allFieldsButIdSetToExcludedField,
        columnNames: columnNames,
    }
}

type QueryTemplates struct {
    mergeTables StringsByDatabase
    createTempTable StringsByDatabase
    createTable StringsByDatabase
    selectQuery string
    ColumnNames []string
}

func createQueryTemplates(t reflect.Type, fieldTypesStrings StringsByDatabase) QueryTemplates {
    helperStrings := createHelperForType(t)

    createTempTable := StringsByDatabase{
        Postgres: "CREATE TEMP TABLE %s(" + fieldTypesStrings.Postgres + ")",
        SqlServer: "CREATE TABLE #%s(" + fieldTypesStrings.SqlServer + ")",
    }

    mergeTables := StringsByDatabase{
        // NOTE: 
        // No manual locking is necessary, because this script
        // is the only one touching these two tables, ever.
        Postgres: `
        WITH cte AS (
            DELETE FROM %[1]s
            WHERE id NOT IN (SELECT id FROM %[2]s))
        INSERT INTO %[1]s
        TABLE %[2]s
        ON CONFLICT (id) DO UPDATE SET ` + helperStrings.allFieldsButIdSetToExcludedField + `;
        `,
        SqlServer: `
        MERGE %[1]s AS target
        USING (SELECT * FROM #%[2]s) AS source
        ON source.id = target.id

        WHEN NOT MATCHED BY source THEN DELETE

        WHEN NOT MATCHED BY target THEN
        INSERT (` + helperStrings.allFields + `)
        VALUES (` + helperStrings.allFieldsPrefixedBySource + `)

        WHEN MATCHED THEN
        UPDATE SET ` + helperStrings.allFieldsButIdSetToSourceField + `;`,
    }

    createTables := StringsByDatabase{
        Postgres: `
        DO
        $$
        BEGIN
            IF NOT EXISTS (
                SELECT * FROM pg_catalog.pg_class c
                JOIN   pg_catalog.pg_namespace n ON n.oid = c.relnamespace
                WHERE 
                    -- n.nspname = 'schema_name'
                    -- AND
                    c.relname = '%[1]s'
            )
            THEN CREATE TABLE %[1]s (
            ` + fieldTypesStrings.Postgres + `
            );
            END IF;
        END
        $$;
        `,
        SqlServer: `
        if not exists (
            select 1
            from information_schema.tables
            where table_name = '%[1]s'
        ) begin create table %[1]s (
        ` + fieldTypesStrings.SqlServer +
        `);
        end
        `,
    }

    selectQuery := "SELECT " + helperStrings.allFields + " FROM %s ORDER BY id"

    return QueryTemplates{
        mergeTables: mergeTables,
        createTempTable: createTempTable,
        createTable: createTables,
        selectQuery: selectQuery,
        ColumnNames: helperStrings.columnNames,
    }
}

func (templates *QueryTemplates) TempTable(databaseType DatabaseType, tempTableName string) string {
    template := templates.createTempTable.Get(databaseType)
    r := fmt.Sprintf(template, tempTableName)
    return r
}

type SourceAndTarget struct {
    Source string;
    Target string;
}

func (templates *QueryTemplates) MergeTables(databaseType DatabaseType, p SourceAndTarget) string {
    template := templates.mergeTables.Get(databaseType)
    r := fmt.Sprintf(template, p.Target, p.Source)
    return r
}

func (templates *QueryTemplates) CreateTable(databaseType DatabaseType, tableName string) string {
    template := templates.createTable.Get(databaseType)
    r := fmt.Sprintf(template, tableName)
    return r
}

func (templates *QueryTemplates) Select(tableName string) string {
    r := fmt.Sprintf(templates.selectQuery, tableName)
    return r
}

func assertValidTableName(tableName string) {
    if strings.Contains(tableName, " ") {
        panic("Invalid table name " + tableName)
    }
}

// The table names are lowercased automatically in postgres irrespective of what you pass in.
// The issue is that the table existence check is done against the name and IS case sensitive.
func makeDatabaseTableName(databaseType DatabaseType, suggestedName string) string {
    tableName := suggestedName
    assertValidTableName(tableName)

    if databaseType == Postgres {
        tableName = strings.ToLower(tableName)
    }

    return tableName
}

var ClientTemplates = createQueryTemplates(reflect.TypeOf(Client{}), StringsByDatabase{
    Postgres: `
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    nume VARCHAR(255),
    prenume VARCHAR(255)
    `,
    SqlServer: `
    id INT PRIMARY KEY,
    email NVARCHAR(255) UNIQUE,
    nume NVARCHAR(255),
    prenume NVARCHAR(255)
    `,
})
