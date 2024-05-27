package copyscript

import (
	"bytes"
	"container/list"
	"database/sql"
	"errors"
	"fmt"

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

type NamedConnection struct {
	Connection *sql.DB
	Name       string
	Type       DatabaseType
}

type DatabaseConnectionsContext struct {
	MainDatabaseIndex   int
	Connections         []NamedConnection
}

func (context *DatabaseConnectionsContext) MainDatabase() *NamedConnection {
	return &context.Connections[context.MainDatabaseIndex]
}

func (context *DatabaseConnectionsContext) Destroy() {
    closeConnections(context.Connections)
}

type DatabaseTransactionsContext struct {
    DbContext *DatabaseConnectionsContext
    Transactions []*sql.Tx
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

func (context *DatabaseConnectionsContext) OpenTransactions() (DatabaseTransactionsContext, error) {
    connections := context.Connections

    transactions := make([]*sql.Tx, len(connections))
    for i := range connections {
        connection := &connections[i]
        var err error
        transactions[i], err = connection.Connection.Begin()
        if err != nil {
            closeTransactions(transactions[0 : i])
            return DatabaseTransactionsContext{}, err
        }
    }

    return DatabaseTransactionsContext{
        Transactions: transactions,
        DbContext: context,
    }, nil
}


var ErrMainDbNotFound = errors.New("main database not found")

func closeConnections(slice []NamedConnection) {
    for i := range slice {
        slice[i].Connection.Close()
    }
}

func establishConnectionsFromConfig(config *viper.Viper) (DatabaseConnectionsContext, error) {
	var databaseInfos []ConnectionInfo
	{
		err := config.UnmarshalKey("databases", &databaseInfos)
		if err != nil {
            return DatabaseConnectionsContext{}, err
		}
	}

	mainDatabaseIndex := getDatabaseIndex(config, databaseInfos)
	if mainDatabaseIndex == -1 {
        return DatabaseConnectionsContext{}, ErrMainDbNotFound
	}

	databaseConnections := make([]NamedConnection, len(databaseInfos))
	for i, databaseInfo := range databaseInfos {
		success, databaseType := getDatabaseType(databaseInfo.Type)
		if !success {
            closeConnections(databaseConnections[0 : i])
			return DatabaseConnectionsContext{}, fmt.Errorf("invalid database type: %s", databaseInfo.Type)
		}
		databaseProviderString := databaseProviderString(databaseType)

		database, err := sql.Open(databaseProviderString, databaseInfo.ConnectionString)
		if err != nil {
            closeConnections(databaseConnections[0 : i])
            return DatabaseConnectionsContext{}, fmt.Errorf("error while opening database connection: %w", err)
		}

		databaseConnections[i] = NamedConnection{
            Connection: database,
            Name: databaseInfo.DatabaseName,
            Type: databaseType,
        }
	}

    return DatabaseConnectionsContext{
        MainDatabaseIndex: mainDatabaseIndex,
        Connections: databaseConnections,
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
