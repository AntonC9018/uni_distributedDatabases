package database

import (
	db "common/database"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
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
    db := &f.DbContext.Databases[index]
    gormDb, err := CreateGormDatabase(db, &f.Config)
    return gormDb, err
}

func (f *GormFactory) CreateWrapError(c *gin.Context, index int) *gorm.DB {
    gormDb, err := f.Create(index)
    if err != nil {
        log.Printf("Error while connecting to db: %v", err)
        c.Error(ConnectionError{})
        return nil
    }

    return gormDb.WithContext(c.Request.Context())
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


func HandleLastError(gormDb *gorm.DB, c *gin.Context) bool {
    if err := gormDb.Error; err != nil {
        log.Printf("An error occured while querying the database %v", err)
        c.Error(ConnectionError{})
        return false
    }
    return true
}
