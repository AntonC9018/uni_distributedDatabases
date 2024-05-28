package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/spf13/viper"
)

type ConnectionInfo struct {
	DatabaseName     string
	Type             string
	ConnectionString string
}

type DatabaseType int

const (
	Postgres DatabaseType = iota
	SqlServer = iota
)

type NamedDatabase struct {
	DB *sql.DB
	Name string
	Type DatabaseType
}

type DatabasesContext struct {
	MainDatabaseIndex   int
	Databases    []NamedDatabase
}

func (context *DatabasesContext) MainDatabase() *NamedDatabase {
	return &context.Databases[context.MainDatabaseIndex]
}

func (context *DatabasesContext) Destroy() {
    closeDatabases(context.Databases)
}

type DatabaseTransactionsContext struct {
    Transactions []*sql.Tx
    Connections []*sql.Conn
}

func (context *DatabaseTransactionsContext) GetTransactionAndConnection(i int) (*sql.Tx, *sql.Conn) {
    return context.Transactions[i], context.Connections[i]
}

func (context *DatabaseTransactionsContext) Rollback() {
    closeTransactions(context.Transactions)
}

type TransactionCommitResult struct {
    Errors []error
    AllTransactionsErrored bool
}

func (result TransactionCommitResult) IsError() bool {
    return result.Errors != nil
}

func (err TransactionCommitResult) Error() string {
	var b bytes.Buffer
    if err.AllTransactionsErrored {
        b.WriteString("all transactions have errored:")
    } else {
        b.WriteString("some transactions have errored, the databases might be in an irreversable inconsistent state:")
    }
	for _, child := range err.Errors {
		b.WriteString("\n\t... ")
        b.WriteString(child.Error())
	}
	return b.String()
}

func (context *DatabaseTransactionsContext) Commit() TransactionCommitResult {
    var errors []error
    for _, tx := range context.Transactions {
        err := tx.Commit()
        if err != nil {
            errors = append(errors, err)
        }
    }
    if (len(errors) == 0) {
        return TransactionCommitResult{}
    }
    return TransactionCommitResult{
        Errors: errors,
        AllTransactionsErrored: len(errors) == len(context.Transactions),
    }
}

func (dbContext *DatabasesContext) OpenTransactions(
    context1 context.Context,
    transactionOptions *sql.TxOptions) (DatabaseTransactionsContext, error) {

    namedConnections := dbContext.Databases

    transactions := make([]*sql.Tx, len(namedConnections))
    connections := make([]*sql.Conn, len(namedConnections))
    for i := range namedConnections {
        pool := namedConnections[i].DB
        var err error
        connections[i], err = pool.Conn(context1)
        if err != nil {
            closeConnections(connections[0 : i])
            closeTransactions(transactions[0 : i])
            return DatabaseTransactionsContext{}, err
        }
        transactions[i], err = connections[i].BeginTx(context1, transactionOptions)
        if err != nil {
            closeConnections(connections[0 : i + 1])
            closeTransactions(transactions[0 : i])
            return DatabaseTransactionsContext{}, err
        }

        log.Printf("Established transaction with %s\n", namedConnections[i].Name)
    }

    return DatabaseTransactionsContext{
        Transactions: transactions,
        Connections: connections,
    }, nil
}


var ErrMainDbNotFound = errors.New("main database not found")

func closeDatabases(slice []NamedDatabase) {
    for _, conn := range slice {
        err := conn.DB.Close()
        if err != nil {
            panic("Connection must be closable")
        }
    }
}

func closeConnections(slice []*sql.Conn) {
    for _, conn := range slice {
        err := conn.Close()
        if err != nil {
            panic("Connection must be closable")
        }
    }
}

func establishConnectionsFromConfig(config *viper.Viper) (DatabasesContext, error) {
	var databaseInfos []ConnectionInfo
	{
		err := config.UnmarshalKey("databases", &databaseInfos)
		if err != nil {
            return DatabasesContext{}, err
		}
	}

	mainDatabaseIndex := getDatabaseIndex(config, databaseInfos)
	if mainDatabaseIndex == -1 {
        return DatabasesContext{}, ErrMainDbNotFound
	}

	databaseConnections := make([]NamedDatabase, len(databaseInfos))
	for i, databaseInfo := range databaseInfos {
		success, databaseType := getDatabaseType(databaseInfo.Type)
		if !success {
            closeDatabases(databaseConnections[0 : i])
			return DatabasesContext{}, fmt.Errorf("invalid database type: %s", databaseInfo.Type)
		}
		databaseProviderString := databaseProviderString(databaseType)

		database, err := sql.Open(databaseProviderString, databaseInfo.ConnectionString)
		if err != nil {
            closeDatabases(databaseConnections[0 : i])
            return DatabasesContext{}, fmt.Errorf("error while opening database connection: %w", err)
		}

		databaseConnections[i] = NamedDatabase{
            DB: database,
            Name: databaseInfo.DatabaseName,
            Type: databaseType,
        }
	}

    return DatabasesContext{
        MainDatabaseIndex: mainDatabaseIndex,
        Databases: databaseConnections,
    }, nil
}

func databaseProviderString(databaseType DatabaseType) string {
	var r string
	switch databaseType {
	case Postgres:
		r = "postgres"
	case SqlServer:
		r = "sqlserver"
	default:
		panic("unreachable")
	}
	return r
}

func getDatabaseType(databaseType string) (bool, DatabaseType) {
	switch databaseType {
	case "postgresql":
		return true, Postgres
	case "sqlserver":
		return true, SqlServer
	default:
		return false, -1
	}
}

func getDatabaseIndex(config *viper.Viper, databaseInfos []ConnectionInfo) int {
	mainDatabaseName := config.GetString("MainDatabase")
	if mainDatabaseName == "" {
		return 0
	}
	// find index of main database
	result := -1
	for i := range databaseInfos {
		if databaseInfos[i].DatabaseName == mainDatabaseName {
			result = i
		}
	}
	return result
}
