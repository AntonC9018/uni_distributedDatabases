package copyscript

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"runtime/trace"

	"github.com/spf13/viper"
	"golang.org/x/text/unicode/rangetable"

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
    if !isValidTableName(tableName) {
        panic("Invalid table name " + tableName)
    }
}

func doTheCopying(dbContext *DatabaseConnectionsContext) (err error) {
    // Copy from all databases to all others (star topology).
    // Create new tables if they don't exist.
    // Partition the tables just like in the other db (so we want the command here huh).
    // Copy the data by using one of the methods (see the doc).
    // The tables Foaie and Client are the ones to be copied.
    connections := dbContext.Connections

    transactionContext, err := dbContext.OpenTransactions()
    defer func() {
        if err != nil {
            transactionContext.Rollback()
        }
    }()

    for copyFromIndex := range connections {
        connectionFrom := &connections[copyFromIndex]

        sourceTableName := "Client"
        assertValidTableName(sourceTableName)

        for otherIndex := range connections {
            if (otherIndex == copyFromIndex) {
                continue
            }

            connectionTo := &connections[otherIndex]

            targetTableName := fmt.Sprintf("%s_%s", sourceTableName, connectionTo.Name)
            assertValidTableName(targetTableName)

            // Run script to create targetTableName if it doesn't exist
            // Ideally, this should check the types in the other db maybe? But that's complicated.
            maybeCreateTableQueryString := getMaybeCreateTableQuery(connectionFrom, targetTableName)

            var createTableResult sql.Result
            createTableResult, err = connectionFrom.Connection.Exec(maybeCreateTableQueryString)
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
            readCursor, err = connectionFrom.Connection.Query(selectQuery)
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

        for readCursor.Next() {
            err = readCursor.Scan(scanParams...)
            if err != nil {
                return
            }

            for otherIndex := range connections {
                if otherIndex == copyFromIndex {
                    continue
                }

                connectionTo := connections[otherIndex].Connection
                transactionTo := transactionContext.Transactions[otherIndex]
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
}

func getMaybeCreateTableQueryTemplate(connectionType DatabaseType) string {
	switch connectionType {
	case Postgres:
		return `
            if not exists (
                select 1
                from information_schema.tables
                where table_name = '%[1]s'
            ) then
            CREATE TABLE %[1]s (
                id SERIAL PRIMARY KEY,
                email VARCHAR(255) UNIQUE,
                nume VARCHAR(255),
                prenume VARCHAR(255)
            );
            end if;
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

func getMaybeCreateTableQuery(connectionFrom *NamedConnection, targetTableName string) string {
    template := getMaybeCreateTableQueryTemplate(connectionFrom.Type)
	result := fmt.Sprintf(template, targetTableName)
	return result
}

func main() {
    if mainWithError() != nil {
        os.Exit(-1)
    }
}

func mainWithError() error {
    var config *viper.Viper
    {
        var err error
        config, err = readConfig()
        if err != nil {
            _, configFileNotFound := err.(viper.ConfigFileNotFoundError)
            if configFileNotFound {
                log.Fatalf("Config file not found: %v\n", err)
            } else {
                log.Fatalf("Error while reading config file: %v\n", err)
            }
            return err
        }
    }

    var dbContext DatabaseConnectionsContext
    {
        var err error
        dbContext, err = establishConnectionsFromConfig(config)
        if err != nil {
            log.Fatalf("Error while establishing connections: %v\n", err);
            return err
        }
    }
    defer func() {
        dbContext.Destroy()
    }()

    {
        err := doTheCopying(&dbContext)
        if err != nil {
            log.Fatalf("Error while doing the copying: %v\n", err);
            return err
        }
    }
    return nil
}

