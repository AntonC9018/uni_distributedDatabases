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

type Partition struct {
    Values []string
    SchemeName string
    FuncName string
}

type TypedColumn struct {
    Name string
    Type StringsByDatabase
}

type PartitionedColumn struct {
    Name string
    Partition Partition
}

type templateConfig struct {
    Columns []TypedColumn
    PartitionedColumns []PartitionedColumn
}

type ConcatForDatabaseParams struct {
    Template templateConfig
    DatabaseType DatabaseType
    MakePrimaryKey bool
    Builder *strings.Builder
}

func concatForDatabase(p *ConcatForDatabaseParams) {
    var b = p.Builder
    for i, field := range p.Template.Columns {
        if i != 0 {
            b.WriteString(", ")
        }
        b.WriteString(field.Name)
        b.WriteString(" ")
        b.WriteString(field.Type.Get(p.DatabaseType))
        if i == 0 && p.MakePrimaryKey {
            b.WriteString(" PRIMARY KEY")
        }
    }
}

func concatPrimaryKeys(p *ConcatForDatabaseParams) {
    var sb = p.Builder
    sb.WriteString(", PRIMARY KEY (id")
    for _, partitionedColumn := range p.Template.PartitionedColumns {
        sb.WriteString(", ")
        sb.WriteString(partitionedColumn.Name)
    }
    sb.WriteString(")")
}

func createQueryTemplates(t reflect.Type, config templateConfig) QueryTemplates {
    helperStrings := createHelperForType(t)

    var sb strings.Builder

    var createTempTable StringsByDatabase
    {
        context := ConcatForDatabaseParams{
            Template: config,
            MakePrimaryKey: false,
            Builder: &sb,
        }

        context.DatabaseType = Postgres
        concatForDatabase(&context)
        createTempTable.Postgres = "CREATE TEMP TABLE %s(" + sb.String() + ")"

        sb.Reset()

        context.DatabaseType = SqlServer
        concatForDatabase(&context)
        createTempTable.SqlServer = "CREATE TABLE #%s(" + sb.String() + ")"

        sb.Reset()
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

    var createTables StringsByDatabase
    {
        concatContext := ConcatForDatabaseParams{
            Template: config,
            MakePrimaryKey: false,
            DatabaseType: Postgres,
            Builder: &sb,
        }
        {
            concatContext.DatabaseType = Postgres
            defer sb.Reset()

            sb.WriteString(`
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
                THEN
                CREATE TABLE %[1]s (`)

            concatForDatabase(&concatContext)
            concatPrimaryKeys(&concatContext)

            sb.WriteString(") ")

            if len(config.PartitionedColumns) > 0 {
                sb.WriteString("PARTITION BY LIST (")
                for i, partitionedColumn := range config.PartitionedColumns {
                    if i != 0 {
                        sb.WriteString(", ")
                    }
                    sb.WriteString(partitionedColumn.Name)
                }
            }

            sb.WriteString(";")

            if len(config.PartitionedColumns) > 1 {
                panic("Unimplemented")
            }
            for _, partitionedColumn := range config.PartitionedColumns {
                for _, val := range partitionedColumn.Partition.Values {
                    sb.WriteString("CREATE TABLE ")
                    sb.WriteString("%[1]s_")
                    sb.WriteString(strings.ToLower(val))
                    sb.WriteString(" PARTITION OF %[1]s FOR VALUES IN ('")
                    sb.WriteString(val)
                    sb.WriteString("');")
                }
            }

            sb.WriteString(`
                END IF;
            END
            $$;
            `)

            createTables.Postgres = sb.String()
        }
        {
            concatContext.DatabaseType = SqlServer
            defer sb.Reset()

            sb.WriteString(`
            if not exists (
                select 1
                from information_schema.tables
                where table_name = '%[1]s'
            ) begin
            create table %[1]s (
            `)

            concatForDatabase(&concatContext)
            concatPrimaryKeys(&concatContext)

            sb.WriteString(`)`)

            sb.WriteString("ON ")

            // IDK if this is right, or if the parition scheme is on all the columns
            for i, partitionedColumn := range config.PartitionedColumns {
                if i != 0 {
                    sb.WriteString(", ")
                }
                sb.WriteString(partitionedColumn.Partition.SchemeName)
                sb.WriteString("(")
                sb.WriteString(partitionedColumn.Name)
                sb.WriteString(")")
            }

            sb.WriteString(`
            end
            `)

            createTables.SqlServer = sb.String()
        }
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

var idType = StringsByDatabase{
    Postgres: "INT NOT NULL",
    SqlServer: "INT NOT NULL",
}
var stringType = StringsByDatabase{
    Postgres: "VARCHAR(255) NOT NULL",
    SqlServer: "NVARCHAR(255) NOT NULL",
}
var uniqueStringType = StringsByDatabase{
    Postgres: "VARCHAR(255) NOT NULL UNIQUE",
    SqlServer: "NVARCHAR(255) NOT NULL UNIQUE",
}
var ClientTemplates = createQueryTemplates(reflect.TypeOf(Client{}), templateConfig{
    Columns: []TypedColumn{
        {
            Name: "id",
            Type: idType,
        },
        {
            Name: "email",
            Type: uniqueStringType,
        },
        {
            Name: "nume",
            Type: stringType,
        },
        {
            Name: "prenume",
            Type: stringType,
        },
    },
})

var moneyType = StringsByDatabase{
    SqlServer: "MONEY NOT NULL",
    Postgres: "MONEY NOT NULL",
}
var boolType = StringsByDatabase{
    SqlServer: "BIT NOT NULL",
    Postgres: "BOOLEAN NOT NULL",
}
var ListTemplates = createQueryTemplates(reflect.TypeOf(Foaie{}), templateConfig{
    Columns: []TypedColumn{
        {
            Name: "id",
            Type: idType,
        },
        {
            Name: "pret",
            Type: moneyType,
        },
        {
            Name: "providedTransport",
            Type: boolType,
        },
        {
            Name: "hotel",
            Type: stringType,
        },
        {
            Name: "tip",
            Type: StringsByDatabase{
                Postgres: "foaietip NOT NULL",
                SqlServer: "VARCHAR(10) NOT NULL",
            },
        },
    },
    PartitionedColumns: []PartitionedColumn{
        {
            Name: "tip",
            Partition: Partition{
                Values: []string{ "Munte", "Mare", "Excursie" },
                SchemeName: "tipPartitionScheme",
                FuncName: "tipPartitionFunc",
            },
        },
    },
})
