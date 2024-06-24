package stuff

import (
	"common/config"
	database_config "common/database"
	"webapp/database"

	"gorm.io/gorm"
)

type ApplicationContext struct {
    DbContext database_config.DatabasesContext
    GormFactory database.GormFactory
}

func (c *ApplicationContext) IsDevelopment() bool {
    return isDevelopment()
}

func CreateApplicationContext() (ret ApplicationContext, err error) {
    var dbContext database_config.DatabasesContext
    config_, err := config.ReadConfig();
    if err != nil {
        return
    }

    dbContext, err = database_config.EstablishConnectionsFromConfig(config_)
    if err != nil {
        return
    }

    gormFactory := database.GormFactory{
        Config: gorm.Config{
            NamingStrategy: &database.MyNamer,
        },
        DbContext: &dbContext,
    }

    gormDb, err := gormFactory.Create(dbContext.MainDatabaseIndex)
    _ = gormDb
    if err != nil {
        return
    }

    ret = ApplicationContext{
        DbContext: dbContext,
        GormFactory: gormFactory,
    }
    return
}
