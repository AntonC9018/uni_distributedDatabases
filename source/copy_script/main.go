package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/spf13/viper"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/lib/pq"
)

func readConfig() (*viper.Viper, error) {
	config := viper.New()
	config.SetConfigName("configuration")
	config.SetConfigType("json")
	config.AddConfigPath(".")
	{
		err := config.ReadInConfig()
		if err != nil {
            return config, err
		}
	}
    return config, nil
}

func closeTransactions(transactions []*sql.Tx) {
    for _, tx := range transactions {
        tx.Rollback()
    }
}

type CompositeTransactionError struct {
    Errors []error
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

type ConnectionHelper struct {
    Transaction *sql.Tx;
    Connection *sql.Conn;
    Database *NamedDatabase;
}

func createConnectionHelper(
    dbContext *DatabasesContext,
    transactionContext *DatabaseTransactionsContext,
    index int) ConnectionHelper {

    return ConnectionHelper{
        Transaction: transactionContext.Transactions[index],
        Connection: transactionContext.Connections[index],
        Database: &dbContext.Databases[index],
    }
}

type InitiatedCopyingContext struct {
    tempTableName string
    targetTableName string
    preparedBulk PreparedBulkContext
}

func (c *InitiatedCopyingContext) IsEmpty() bool {
    return len(c.tempTableName) == 0
}

func (c *InitiatedCopyingContext) clearTempTablesForReuse(helper *ConnectionHelper, backgroundContext context.Context) error {
    switch (helper.Database.Type) {
    case Postgres:
        // c.preparedBulk.PostgresStatement.Close()
        queryString := fmt.Sprintf("DELETE * FROM %s", c.tempTableName)
        _, err := helper.Transaction.ExecContext(backgroundContext, queryString)
        return err
    case SqlServer:
        queryString := fmt.Sprintf("DELETE * FROM #%s", c.tempTableName)
        _, err := helper.Transaction.ExecContext(backgroundContext, queryString)
        return err
    default:
        panic("unreachable")
    }
}

type ClearAllTempTablesForReuseParams struct {
    CopyingContexts []InitiatedCopyingContext
    IterationHelper OtherDbIterationHelper
    BackgroundContext context.Context
}

func clearAllTempTablesForReuse(p ClearAllTempTablesForReuseParams) error {

    for i, helper := range p.IterationHelper.Iter() {
        copyingContext := &p.CopyingContexts[i]
        err := copyingContext.clearTempTablesForReuse(&helper, p.BackgroundContext)
        if err != nil {
            return err
            // panic("Connection broken? What in the world happened?")
        }
    }
    return nil
}

type OtherDbIterationHelper struct {
    dbContext *DatabasesContext
    transactionContext *DatabaseTransactionsContext
    fromIndex int
}

// This is not in my version of go yet.
type Seq2[T1 any, T2 any] func(func(T1, T2) bool)

func (context OtherDbIterationHelper) Iter() Seq2[int, ConnectionHelper] {
    return func(body func(int, ConnectionHelper) bool) {
        for i := range context.dbContext.Databases {
            if i == context.fromIndex {
                continue
            }
            helper := createConnectionHelper(context.dbContext, context.transactionContext, i)

            shouldKeepGoing := body(i, helper)
            if !shouldKeepGoing {
                return
            }
        }
    }
}

func doTheCopying(dbContext *DatabasesContext, backgroundContext context.Context) (err error) {
    var transactionOptions = sql.TxOptions{
        // Isolation: sql.LevelSerializable,
        ReadOnly: false,
    }

    var transactionContext DatabaseTransactionsContext
    {
        contextWithTimeout, cancel := context.WithTimeout(backgroundContext, time.Second * 2)
        defer cancel()

        transactionContext, err = dbContext.OpenTransactions(contextWithTimeout, &transactionOptions)

        if err != nil {
            return
        }
    }
    defer func() {
        if err != nil {
            transactionContext.Rollback()
        }
    }()
    log.Println("Transactions opened")

    const tempTablePrefix = "temp"

    copyingContexts := make([]InitiatedCopyingContext, len(dbContext.Databases))

    for copyFromIndex := range dbContext.Databases {
        otherDbIterationHelper := OtherDbIterationHelper{
            dbContext: dbContext,
            transactionContext: &transactionContext,
            fromIndex: copyFromIndex,
        }

        from := createConnectionHelper(dbContext, &transactionContext, copyFromIndex)
        log.Printf("Copying from %s\n", from.Database.Name)

        clientTableName := "Client"
        sourceTableName := makeDatabaseTableName(from.Database.Type, clientTableName)

        for otherIndex, to := range otherDbIterationHelper.Iter() {
            targetTableName := fmt.Sprintf("%s_%s", clientTableName, to.Database.Name)
            targetTableName = makeDatabaseTableName(to.Database.Type, targetTableName)

            // Run script to create targetTableName if it doesn't exist
            // Ideally, this should check the types in the other db maybe? But that's complicated.
            maybeCreateTableQueryString := getMaybeCreateTableQuery(to.Database, targetTableName)

            var createTableResult sql.Result
            fmt.Println(maybeCreateTableQueryString)
            createTableResult, err = to.Transaction.ExecContext(backgroundContext, maybeCreateTableQueryString)
            if err != nil {
                return
            }

            log.Printf("Output table %s created for %s\n", targetTableName, to.Database.Name)

            _ = createTableResult

            // Next, we want to prepare the statements for insertion for all of them.
            tempTableName := fmt.Sprintf("%s_%s", tempTablePrefix, targetTableName)
            tempTableName = makeDatabaseTableName(to.Database.Type, tempTableName)
            var preparedBulk PreparedBulkContext
            preparedBulk, err = prepareInsertIntoTempTableStatement(to, tempTableName, backgroundContext)
            if err != nil {
                return
            }

            log.Printf("Temporary table %s created for %s\n", tempTableName, to.Database.Name)
            
            copyingContexts[otherIndex] = InitiatedCopyingContext{
                preparedBulk: preparedBulk,
                tempTableName: tempTableName,
                targetTableName: targetTableName,
            }
        }

        // Open cursor reading all data from the db
        var readCursor *sql.Rows
        {
            const selectQueryTemplate = "SELECT " + clientFieldNamesString + " FROM %s ORDER BY id"
            selectQuery := fmt.Sprintf(selectQueryTemplate, sourceTableName)
            readCursor, err = from.Transaction.QueryContext(backgroundContext, selectQuery)
            if err != nil {
                return
            }

            log.Printf("Data copying started\n")
        }
        defer func() {
            readCursor.Close()
        }()

        // Buffer things manually, because all of the lower-level code 
        // in the adapter libraries is private.
        // We cannot reuse their memory and their buffers, we'll have to copy.

        // As much as it pains me, this stupid way seems to be the only way
        // to do this without reimplementing everything.

        var obj = Client{}
        const fieldCount = 4
        outputPointersArray := [fieldCount]interface{}{}
        scanParams := copyFieldPointers(&obj, outputPointersArray[:])

        for readCursor.Next() {
            err = readCursor.Scan(scanParams[:]...)
            if err != nil {
                return
            }

            for otherIndex, to := range otherDbIterationHelper.Iter() {
                copyingContext := copyingContexts[otherIndex]

                switch (to.Database.Type) {
                case Postgres:
                    _, err = copyingContext.preparedBulk.PostgresStatement.Exec(outputPointersArray[:]...)
                    if err != nil {
                        return
                    }
                case SqlServer:
                    err = copyingContext.preparedBulk.SqlServerBulk.AddRow(outputPointersArray[:])
                    if err != nil {
                        return
                    }
                }
            }
        }

        err = readCursor.Err()
        if err != nil {
            return
        }

        for otherIndex, to := range otherDbIterationHelper.Iter() {
            copyingContext := copyingContexts[otherIndex]

            switch (to.Database.Type) {
            case Postgres:
                _, err = copyingContext.preparedBulk.PostgresStatement.Exec()
                if err != nil {
                    return
                }
            case SqlServer:
                bulk := copyingContext.preparedBulk.SqlServerBulk
                _, err = bulk.Done()
                if err != nil {
                    return
                }
            }

            // Do the merge
            {
                queryStringTemplate := getMergeClientTablesQueryTemplate(to.Database.Type)
                mergeQueryString := fmt.Sprintf(
                    queryStringTemplate,
                    copyingContext.targetTableName,
                    copyingContext.tempTableName)

                log.Printf("Executing Query:\n%s\n", mergeQueryString)

                var result sql.Result
                result, err = to.Transaction.ExecContext(backgroundContext, mergeQueryString)
                if err != nil {
                    return
                }

                rowsAffected, _ := result.RowsAffected()
                log.Printf("The query affected %d rows", rowsAffected)
            }
        }
    }

    {
        result := transactionContext.Commit()
        if result.IsError() {
            err = result
            return
        }
    }
    return
}

const PostgresClientTableFields = `
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    nume VARCHAR(255),
    prenume VARCHAR(255)
`

const PostgresCreateClientTableQueryTemplate = `
CREATE TABLE %[1]s (
` + PostgresClientTableFields + `
)
`

const SqlServerClientTableFields = `
    id INT PRIMARY KEY,
    email NVARCHAR(255) UNIQUE,
    nume NVARCHAR(255),
    prenume NVARCHAR(255)
`
const SqlServerCreateClientTableQueryTemplate = `
CREATE TABLE %[1]s (
` + SqlServerClientTableFields + `
)
`
func getCreateClientTableQueryTemplate(connectionType DatabaseType) string {
    switch (connectionType) {
    case Postgres:
        return PostgresCreateClientTableQueryTemplate
    case SqlServer:
        return SqlServerCreateClientTableQueryTemplate
    default:
        panic("unreachable")
    }
}

func getMaybeCreateTableQueryTemplate(connectionType DatabaseType) string {
	switch connectionType {
	case Postgres:
		return `
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
            ` + PostgresCreateClientTableQueryTemplate +
            `;
            END IF;
        END
        $$;
        `
	case SqlServer:
        return `
        if not exists (
            select 1
            from information_schema.tables
            where table_name = '%[1]s'
        ) begin
        ` + SqlServerCreateClientTableQueryTemplate +
        `
        end
        `
	default:
		panic("unreachable")
	}
}

func getMaybeCreateTableQuery(database *NamedDatabase, targetTableName string) string {
    template := getMaybeCreateTableQueryTemplate(database.Type)
	result := fmt.Sprintf(template, targetTableName)
	return result
}

func main() {
    log.Println("Application started")
    context1 := context.Background()
    err := mainWithError(context1)
    if err != nil {
        os.Exit(-1)
    }
}

func mainWithError(context context.Context) error {
    
    var config *viper.Viper
    {
        var err error
        config, err = readConfig()
        if err != nil {
            _, configFileNotFound := err.(viper.ConfigFileNotFoundError)
            if configFileNotFound {
                log.Printf("Config file not found: %v\n", err)
            } else {
                log.Printf("Error while reading config file: %v\n", err)
            }
            return err
        }
        log.Printf("Config loaded")
    }

    var dbContext DatabasesContext
    {
        var err error
        dbContext, err = establishConnectionsFromConfig(config)
        if err != nil {
            log.Printf("Error while establishing connections: %v\n", err);
            return err
        }
        log.Printf("Database connections opened")
    }
    defer func() {
        dbContext.Destroy()
    }()

    {
        err := doTheCopying(&dbContext, context)
        if err != nil {
            log.Printf("Error while doing the copying: %v\n", err);
            return err
        }
        log.Printf("Copying done")
    }
    return nil
}

func copyFieldPointers(ptr interface{}, output []interface{}) []interface{} {
    valueInfo := reflect.ValueOf(ptr)
    numFields := valueInfo.Elem().Type().NumField()
    if numFields < len(output) {
        log.Fatalf("The output array should have at least %d elements, got %d", numFields, len(output))
    }
    for i := 0; i < numFields; i++ {
        ptr1 := valueInfo.Elem().Field(i).Addr().Interface()
        output[i] = ptr1
    }
    return output[0 : numFields]
}

func getCreateTempTableQueryTemplate(databaseType DatabaseType) string {
    switch (databaseType) {
    case Postgres:
        return "CREATE TEMP TABLE %s(" + PostgresClientTableFields + ")"
    case SqlServer:
        return "CREATE TABLE #%s(" + SqlServerClientTableFields + ")"
    default:
        panic("unreachable")
    }
}

type PreparedBulkContext struct {
    PostgresStatement *sql.Stmt
    SqlServerBulk *mssql.Bulk
}

func prepareInsertIntoTempTableStatement(
    c ConnectionHelper,
    tempTableName string,
    context context.Context) (PreparedBulkContext, error) {

    // Create temp table
    createTempTableQueryTemplate := getCreateTempTableQueryTemplate(c.Database.Type)
    createTempTableQueryString := fmt.Sprintf(createTempTableQueryTemplate, tempTableName)

    // No need to delete it, it will be deleted automatically.
    // It is removed automatically at the end of the transaction (?).
    _, err := c.Transaction.Exec(createTempTableQueryString)
    if err != nil {
        return PreparedBulkContext{}, err
    }

    switch (c.Database.Type) {
    case Postgres:
        copyInString := pq.CopyIn(tempTableName, clientColumnNames...)
        statement, err := c.Transaction.PrepareContext(context, copyInString)
        return PreparedBulkContext{
            PostgresStatement: statement,
        }, err
    case SqlServer:
        var bulk *mssql.Bulk
        c.Connection.Raw(func(driverConn any) error {
            sqlServerConn := driverConn.(*mssql.Conn)
            bulk = sqlServerConn.CreateBulkContext(context, "#" + tempTableName, clientColumnNames)
            return nil
        })
        return PreparedBulkContext{
            SqlServerBulk: bulk,
        }, nil
    default:
        panic("unreachable")
    }
}


type Client struct {
    Id int32
    Email string
    Nume string
    Prenume string
}
// These either have to stay comptime constants, or we have to cache the templates globally.
const clientFieldNamesString = "id, email, nume, prenume"
const clientFieldNamesStringPrefixedByTarget = "target.id, target.email, target.nume, target.prenume"
const clientFieldNamesStringPrefixedBySource = "source.id, source.email, source.nume, source.prenume"
const clientSetAllFieldsButIdFromSourceString = "email = source.email, nume = source.nume, prenume = source.prenume"
const clientSetAllFieldsButIdFromExcludedString = "email = excluded.email, nume = excluded.nume, prenume = excluded.prenume"
var clientColumnNames = []string { "id", "email", "nume", "prenume" }

func getMergeClientTablesQueryTemplate(databaseType DatabaseType) string {
    switch (databaseType) {
    case Postgres:
        // NOTE: 
        // No manual locking is necessary, because this script
        // is the only one touching these two tables, ever.
        return `
        WITH cte AS (
            DELETE FROM %[1]s
            WHERE id NOT IN (SELECT id FROM %[2]s))
        INSERT INTO %[1]s
        TABLE %[2]s
        ON CONFLICT (id) DO UPDATE SET ` + clientSetAllFieldsButIdFromExcludedString + `;
        `
    case SqlServer:
        return `
        MERGE %[1]s AS target
        USING (SELECT * FROM #%[2]s) AS source
        ON source.id = target.id

        WHEN NOT MATCHED BY source THEN DELETE

        WHEN NOT MATCHED BY target THEN
        INSERT (` + clientFieldNamesString + `)
        VALUES (` + clientFieldNamesStringPrefixedBySource + `)

        WHEN MATCHED THEN
        UPDATE SET ` + clientSetAllFieldsButIdFromSourceString + `;`
    default:
        panic("unreachable")
    }
}
