package database

import (
	db "common/database"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
    _ "github.com/jinzhu/gorm/dialects/postgres"
	"gorm.io/gorm"
)

type GormContext struct {
    GormDatabases []*gorm.DB
    MainDatabaseIndex int
}

func (c *GormContext) MainDatabase() *gorm.DB {
    return c.GormDatabases[c.MainDatabaseIndex]
}

func createDialector(database *db.NamedDatabase) gorm.Dialector {
    switch database.Type {
    case db.Postgres:
        config := postgres.Config{
            Conn: database.DB,
        }
        dialector := postgres.New(config)
        return dialector

    case db.SqlServer:
        config := sqlserver.Config{
            Conn: database.DB,
        }
        dialector := sqlserver.New(config)
        return dialector

    default:
        panic("unreachable")
    }
}

func CreateGormDatabase(database *db.NamedDatabase, gormConfig *gorm.Config) (*gorm.DB, error) {
    dialector := createDialector(database)
    return gorm.Open(dialector, gormConfig)
}


type GormFactory struct {
    Config gorm.Config
    DbContext *db.DatabasesContext
}

func (f *GormFactory) Create(index int) (*gorm.DB, error) {
    gormDb, err := CreateGormDatabase(f.DbContext.MainDatabase(), &f.Config)
    return gormDb, err
}

type ConnectionError struct {
}

func (err ConnectionError) Error() string {
    return "Error while connecting to database"
}


type PaginationError struct {
}

func (err PaginationError) Error() string {
    return "Error while paginating the query"
}
