package main

import (
	conf "common/config"
	db "common/database"
	"common/models"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/spf13/viper"

	pq "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	mssql "github.com/microsoft/go-mssqldb"
)

type ConnectionHelper struct {
    Transaction *sql.Tx;
    Connection *sql.Conn;
    Database *db.NamedDatabase;
}

func createConnectionHelper(
    dbContext *db.DatabasesContext,
    transactionContext *db.DatabaseTransactionsContext,
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

type OtherDbIterationHelper struct {
    dbContext *db.DatabasesContext
    transactionContext *db.DatabaseTransactionsContext
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

type AllTablesToCopyIteratorContext struct {
    CopyingContexts []InitiatedCopyingContext
    allModels models.AllTableModels
    fieldPointers [5]interface{}
    fieldValuePointers [5]interface{}
}

func createAllTablesToCopyIteratorContext(databaseCount int) AllTablesToCopyIteratorContext {

    copyingContexts := make([]InitiatedCopyingContext, databaseCount)
    return AllTablesToCopyIteratorContext{
        CopyingContexts: copyingContexts,
    }
}

type TableToCopyIterator struct {
    Context *AllTablesToCopyIteratorContext
    ModelTypeName string
    ScanParameters []interface{}
    ValueScanParameters []interface{}
    Templates *QueryTemplates
}

func getTemplates(index models.ModelIndex) *QueryTemplates {
    switch (index) {
    case models.ClientIndex:
        return &ClientTemplates
    case models.ListIndex:
        return &ListTemplates
    default:
        panic("unreachable")
    }
}

func (c *AllTablesToCopyIteratorContext) Iter() Seq2[int, TableToCopyIterator] {
    return func(body func(int, TableToCopyIterator) bool) {
        for i := 0; i < models.ModelCount; i++ {
            modelIndex := models.MinIndex + (models.ModelIndex)(i);
            model := c.allModels.Get(modelIndex)
            modelName := reflect.TypeOf(model).Elem().Name()
            if len(modelName) == 0 {
                panic("Model name can't be empty")
            }
            scanParameters := CopyFieldPointers(model, c.fieldPointers[:])
            valueScanParameters := c.fieldValuePointers[0 : len(scanParameters)]
            ValuesFromPointers(scanParameters, valueScanParameters)

            iterator := TableToCopyIterator{
                Context: c,
                ModelTypeName: modelName,
                ScanParameters: scanParameters,
                ValueScanParameters: valueScanParameters,
                Templates: getTemplates(modelIndex),
            }
            shouldKeepGoing := body(i, iterator)
            if !shouldKeepGoing {
                return
            }
        }
    }
}

func doTheCopying(dbContext *db.DatabasesContext, backgroundContext context.Context) (err error) {
    var transactionOptions = sql.TxOptions{
        // Isolation: sql.LevelSerializable,
        ReadOnly: false,
    }

    var transactionContext db.DatabaseTransactionsContext
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

    modelsContext := createAllTablesToCopyIteratorContext(len(dbContext.Databases))

    for _, modelIter := range modelsContext.Iter() {
        for copyFromIndex := range dbContext.Databases {
            otherDbIterationHelper := OtherDbIterationHelper{
                dbContext: dbContext,
                transactionContext: &transactionContext,
                fromIndex: copyFromIndex,
            }

            from := createConnectionHelper(dbContext, &transactionContext, copyFromIndex)
            log.Printf("Copying from %s\n", from.Database.Name)

            sourceTableName := makeDatabaseTableName(from.Database.Type, modelIter.ModelTypeName)

            for otherIndex, to := range otherDbIterationHelper.Iter() {
                targetTableName := fmt.Sprintf("%s_%s", modelIter.ModelTypeName, from.Database.Name)
                targetTableName = makeDatabaseTableName(to.Database.Type, targetTableName)

                // Run script to create targetTableName if it doesn't exist
                // Ideally, this should check the types in the other db maybe? But that's complicated.
                createTableQueryString := modelIter.Templates.CreateTable(to.Database.Type, targetTableName)

                var createTableResult sql.Result
                fmt.Println(createTableQueryString)
                createTableResult, err = to.Transaction.ExecContext(backgroundContext, createTableQueryString)
                if err != nil {
                    return
                }

                log.Printf("Output table %s created for %s\n", targetTableName, to.Database.Name)

                _ = createTableResult

                // Next, we want to prepare the statements for insertion for all of them.
                tempTableName := fmt.Sprintf("%s_%s", tempTablePrefix, targetTableName)
                tempTableName = makeDatabaseTableName(to.Database.Type, tempTableName)
                var preparedBulk PreparedBulkContext
                preparedBulk, err = prepareInsertIntoTempTableStatement(to, modelIter.Templates, tempTableName, backgroundContext)
                if err != nil {
                    return
                }

                log.Printf("Temporary table %s created for %s\n", tempTableName, to.Database.Name)
                
                modelIter.Context.CopyingContexts[otherIndex] = InitiatedCopyingContext{
                    preparedBulk: preparedBulk,
                    tempTableName: tempTableName,
                    targetTableName: targetTableName,
                }
            }

            // Open cursor reading all data from the db
            var readCursor *sql.Rows
            {
                selectQuery := modelIter.Templates.Select(sourceTableName)
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

            for readCursor.Next() {
                scanParams := modelIter.ScanParameters
                err = readCursor.Scan(scanParams[:]...)
                if err != nil {
                    return
                }

                for otherIndex, to := range otherDbIterationHelper.Iter() {
                    copyingContext := &modelIter.Context.CopyingContexts[otherIndex]
                    fieldValues := modelIter.ValueScanParameters

                    switch (to.Database.Type) {
                    case db.Postgres:
                        // Yikes, we have to box every single field.
                        // There isn't really an easy way around this...
                        // I really wanted the library to handle batching on its own.
                        row := make([]any, len(fieldValues))
                        BoxValues(scanParams, row)
                        p := &copyingContext.preparedBulk.Postgres
                        p.Fields = append(p.Fields, row)
                    case db.SqlServer:
                        err = copyingContext.preparedBulk.SqlServerBulk.AddRow(fieldValues[:])
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
                copyingContext := &modelIter.Context.CopyingContexts[otherIndex]

                switch (to.Database.Type) {
                case db.Postgres:
                    f := copyingContext.preparedBulk.Postgres.Fields
                    to.Connection.Raw(func(driverConn any) error {
                        conn := driverConn.(*stdlib.Conn)
                        _, err := conn.Conn().CopyFrom(
                            backgroundContext,
                            pq.Identifier{copyingContext.tempTableName},
                            modelIter.Templates.ColumnNames,
                            pq.CopyFromRows(f))
                        return err
                    })
                    if err != nil {
                        return
                    }
                case db.SqlServer:
                    bulk := copyingContext.preparedBulk.SqlServerBulk
                    _, err = bulk.Done()
                    if err != nil {
                        return
                    }
                }

                // Do the merge
                {
                    mergeQueryString := modelIter.Templates.MergeTables(to.Database.Type, SourceAndTarget{
                        Source: copyingContext.tempTableName,
                        Target: copyingContext.targetTableName,
                    })

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
        config, err = conf.ReadConfig()
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

    var dbContext db.DatabasesContext
    {
        var err error
        dbContext, err = db.EstablishConnectionsFromConfig(config)
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

type PostgresBulk struct {
    // Just copies of objects
    Fields [][]any
}

type PreparedBulkContext struct {
    Postgres PostgresBulk 
    SqlServerBulk *mssql.Bulk
}

func prepareInsertIntoTempTableStatement(
    c ConnectionHelper,
    templates *QueryTemplates,
    tempTableName string,
    context context.Context) (PreparedBulkContext, error) {

    createTempTableQueryString := templates.TempTable(c.Database.Type, tempTableName)

    // No need to delete it, it will be deleted automatically.
    // It is removed automatically at the end of the transaction (?).
    _, err := c.Transaction.Exec(createTempTableQueryString)
    if err != nil {
        return PreparedBulkContext{}, err
    }

    switch (c.Database.Type) {
    case db.Postgres:
        return PreparedBulkContext{
            Postgres: PostgresBulk{},
        }, err
    case db.SqlServer:
        var bulk *mssql.Bulk
        c.Connection.Raw(func(driverConn any) error {
            sqlServerConn := driverConn.(*mssql.Conn)
            bulk = sqlServerConn.CreateBulkContext(context, "#" + tempTableName, templates.ColumnNames)
            return nil
        })
        return PreparedBulkContext{
            SqlServerBulk: bulk,
        }, nil
    default:
        panic("unreachable")
    }
}

