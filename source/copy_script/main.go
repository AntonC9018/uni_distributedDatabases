package main

import (
	"github.com/spf13/viper"

	"database/sql"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/lib/pq"

	"fmt"
)

type ConnectionInfo struct {
	DatabaseName     string
	Type             string
	ConnectionString string
}

type DatabaseType int

const (
	Postgres DatabaseType = iota
	SqlServer
)

type NamedConnection struct {
	Connection *sql.DB
	Name       string
	Type       DatabaseType
}

type DatabaseConnectionsContext struct {
	MainDatabaseIndex   int
	PostgresConnections []NamedConnection
}

func (context *DatabaseConnectionsContext) MainDatabase() *NamedConnection {
	return &context.PostgresConnections[context.MainDatabaseIndex]
}

func main() {
	config := viper.New()
	config.SetConfigName("configuration")
	config.SetConfigType("json")
	config.AddConfigPath(".")
	{
		err := config.ReadInConfig()
		if err != nil {
			_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
			if configFileNotFound {
				panic(fmt.Errorf("config file not found: %w", err))
			} else {
				panic(fmt.Errorf("error while reading config file: %w", err))
			}
		}
	}

	var databaseInfos []ConnectionInfo
	{
		err := config.UnmarshalKey("databases", &databaseInfos)
		if err != nil {
			panic(fmt.Errorf("error while reading out database connection strings: %w", err))
		}
	}

	mainDatabaseIndex := getDatabaseIndex(config, databaseInfos)
	if mainDatabaseIndex == -1 {
		panic(fmt.Errorf("main database not found"))
	}

	databaseConnections := make([]*sql.DB, len(databaseInfos))
	for i := range databaseInfos {
		success, databaseType := getDatabaseType(databaseInfos[i].Type)
		if !success {
			panic(fmt.Errorf("invalid database type: %s", databaseInfos[i].Type))
		}

		var databaseProviderString string
		switch databaseType {
		case Postgres:
			databaseProviderString = "postgres"
		case SqlServer:
			databaseProviderString = "sqlserver"
		default:
			// unreachable
			panic("unreachable")
		}

		database, err := sql.Open(databaseProviderString, databaseInfos[i].ConnectionString)
		if err != nil {
			panic(fmt.Errorf("error while opening database connection: %w", err))
		}

		databaseConnections[i] = database
	}

	fmt.Println("All seems to be good!")
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
