package copyscript

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

func main() {
    var config *viper.Viper
    {
        var err error
        config, err = readConfig()
        if err != nil {
            _, configFileNotFound := err.(viper.ConfigFileNotFoundError)
            if configFileNotFound {
                log.Fatalf("config file not found: %v\n", err)
            } else {
                log.Fatalf("error while reading config file: %v\n", err)
            }
            os.Exit(-1)
        }
    }

    var dbContext DatabaseConnectionsContext
    {
        var err error
        dbContext, err = establishConnectionsFromConfig(config)
        if err != nil {
            log.Fatalf("Error while establishing connections: %v \n");
            os.Exit(-1)
        }
    }


    // Copy from all databases to all others (star).
    // Create new tables if they don't exist.
    // Partition the tables just like in the other db (so we want the command here huh).
    // Copy the data by using one of the methods (see the doc).
    // The tables Foaie and Client are the ones to be copied.
    connections := dbContext.Connections

    // Open a transaction for all of these.
    transactions := make([]*sql.Tx, len(connections))
    for i := range connections {
        connection := &connections[i]
        tx, err := connection.Connection.Begin()
        if err != nil {
            log.Fatalf("Error while opening transaction: %v \n", err)
            closeTransactions(transactions[0 : i])
            os.Exit(-1)
        }

        transactions[i] = tx
    }

    for copyFromIndex := range connections {
        connectionFrom := &connections[copyFromIndex]

        for copyToIndex := range connections {
            if (copyToIndex == copyFromIndex) {
                continue
            }

            connectionTo := &connections[copyToIndex]
            sourceTableName := "Client"
            targetTableName := fmt.Sprintf("%s_%s", sourceTableName, connectionTo.Name)

            // Run script to create targetTableName if it doesn't exist
            // Ideally, this should check the types in the other db maybe? But that's complicated.
            var maybeCreateTableQueryTemplate string 
            switch (connectionFrom.Type) {
                case Postgres:
                    maybeCreateTableQueryTemplate = `
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
                    maybeCreateTableQueryTemplate = `
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
            maybeCreateTableQueryString := fmt.Sprintf(maybeCreateTableQueryTemplate, targetTableName)

            result, err := connectionFrom.Connection.Exec(maybeCreateTableQueryString)
            if err == nil {

            }
        }
    }
}

