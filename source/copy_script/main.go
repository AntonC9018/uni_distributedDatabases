package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/lib/pq"
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

    for copyFromIndex := range dbContext.Databases {
        from := createConnectionHelper(dbContext, &transactionContext, copyFromIndex)
        log.Printf("Copying from %s\n", from.Database.Name)

        sourceTableName := "Client"
        assertValidTableName(sourceTableName)

        for otherIndex := range transactionContext.Connections {
            if (otherIndex == copyFromIndex) {
                continue
            }

            to := createConnectionHelper(dbContext, &transactionContext, otherIndex)

            targetTableName := fmt.Sprintf("%s_%s", sourceTableName, to.Database.Name)
            assertValidTableName(targetTableName)

            // Run script to create targetTableName if it doesn't exist
            // Ideally, this should check the types in the other db maybe? But that's complicated.
            maybeCreateTableQueryString := getMaybeCreateTableQuery(from.Database, targetTableName)
            fmt.Println(maybeCreateTableQueryString)

            var createTableResult sql.Result
            createTableResult, err = from.Transaction.ExecContext(backgroundContext, maybeCreateTableQueryString)
            if err != nil {
                return
            }

            _ = createTableResult
        }

        const fieldNamesString = "id, email, nume, prenume"

        // Open cursor reading all data from the db
        var readCursor *sql.Rows
        {
            const selectQueryTemplate = "SELECT " + fieldNamesString + " FROM %s ORDER BY id"
            selectQuery := fmt.Sprintf(selectQueryTemplate, sourceTableName)
            readCursor, err = from.Transaction.QueryContext(backgroundContext, selectQuery)
            if err != nil {
                return
            }
        }
        defer func() {
            readCursor.Close()
        }()

        const fieldCount = 4
        var bytesByField = make([]sql.RawBytes, fieldCount)
        var scanParams = make([]interface{}, fieldCount)
        {
            for i := range bytesByField {
                scanParams[i] = &bytesByField[i]
            }
        }

        err = from.Connection.Raw(func(driverConn any) error {
            fmt.Printf("Conn: %T\n", driverConn)
            return nil
        })
        for readCursor.Next() {
            err = readCursor.Scan(scanParams...)
            if err != nil {
                return
            }

            for otherIndex := range dbContext.Databases {
                if otherIndex == copyFromIndex {
                    continue
                }

                to := createConnectionHelper(dbContext, &transactionContext, otherIndex)
                _ = to
            }
        }

        err = readCursor.Err()
        if err != nil {
            return
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

func getMaybeCreateTableQueryTemplate(connectionType DatabaseType) string {
	switch connectionType {
	case Postgres:
		return `
        DO
        $$
        BEGIN
            IF NOT EXISTS (
                SELECT 1
                FROM pg_catalog.pg_tables
                WHERE schemaname = 'public' AND tablename = '%[1]s'
            ) THEN
            CREATE TABLE %[1]s (
                id SERIAL PRIMARY KEY,
                email VARCHAR(255) UNIQUE,
                nume VARCHAR(255),
                prenume VARCHAR(255)
            );
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
            CREATE TABLE %[1]s (
                id INT PRIMARY KEY,
                email NVARCHAR(255) UNIQUE,
                nume NVARCHAR(255),
                prenume NVARCHAR(255)
            );
            end;
        `
	default:
		panic("unreachable")
	}
}

func getMaybeCreateTableQuery(connectionFrom *NamedDatabase, targetTableName string) string {
    template := getMaybeCreateTableQueryTemplate(connectionFrom.Type)
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

