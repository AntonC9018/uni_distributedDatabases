package create

import (
	"fmt"
	"webapp/database"
	"webapp/stuff"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

    clientmod "common/models/client"
)

type CreateClientDto struct {
    Email string `form:"email"`
    Nume string `form:"nume"`
    Prenume string `form:"prenume"`
}

func getClientMask(client *database.Client) clientmod.FieldMask {
    ret := clientmod.FieldMask{}
    if client.Nume != "" {
        ret.SetNume(true)
    }
    if client.Prenume != "" {
        ret.SetPrenume(true)
    }
    return ret
}

func pushClientData(
    gormDb *gorm.DB,
    client *database.Client,
    maskToSet clientmod.FieldMask) {

    updateColumnNames := make([]string, 0, maskToSet.Count())
    if maskToSet.Nume() {
        updateColumnNames = append(updateColumnNames, "nume")
    }
    if maskToSet.Prenume() {
        updateColumnNames = append(updateColumnNames, "prenume")
    }

    gormDb.Clauses(clause.OnConflict{
        Columns: []clause.Column{{ Name: "email" }},
        DoUpdates: clause.AssignmentColumns(updateColumnNames),
    })
    gormDb.Create(client)
}

func bindDto(c *gin.Context) (ret CreateClientDto) {
    err := c.Bind(&ret)
    if err != nil {
        c.Error(err)
        return
    }

    err = validation.ValidateStruct(
        &ret,
        validation.Field(
            &ret.Email,
            validation.Required,
            is.Email),
        validation.Field(&ret.Nume, is.Alphanumeric),
        validation.Field(&ret.Prenume, is.Alphanumeric))
    if err != nil {
        c.Error(err)
    }

    return
}

// type handleResult int

func handle(c *gin.Context, appContext *stuff.ApplicationContext) {
    errScope := stuff.CreateErrorScope(c)
    dto := bindDto(c)
    if errScope.HasErrors() {
        return
    }

    // First look in the main database
    dbIndex := appContext.DbContext.MainDatabaseIndex
    gormDb := appContext.GormFactory.CreateWrapError(c, dbIndex)
    if errScope.HasErrors() {
        return
    }

    tx := gormDb.Begin()
    tx.Model(&database.Client{})
    client := database.Client{}
    tx.Where("email = ?", dto.Email)

    {
        client.Email = dto.Email
        client.Nume = dto.Nume
        client.Prenume = dto.Prenume
    }

    alreadyExistsMask := getClientMask(&client)
    const updatedFieldCount = 2

    if alreadyExistsMask.Empty() {
        tx.First(&client)

        isNotFound := tx.Error == gorm.ErrRecordNotFound
        if !isNotFound {
            database.HandleLastError(tx, c)
        }
    } else {
        pushClientData(tx, &client, alreadyExistsMask)
        database.HandleLastError(tx, c)
    }

    if errScope.HasErrors() {
        return
    }

    alreadyExistsMask = getClientMask(&client)
    if alreadyExistsMask.Count() == updatedFieldCount {
        return
    }

    // find the values for the other fields in the other databases
    {
        dbs := appContext.DbContext.Databases

        // TODO: Multithread
        for i := range dbs {
            if i == dbIndex {
                continue
            }

            otherGormDb := appContext.GormFactory.CreateWrapError(c, i)
            if otherGormDb == nil {
                continue
            }

            o := otherGormDb
            o.Model(&database.Client{})
            o.Where("email = ?", dto.Email)
            result := database.Client{}
            o.First(&result)

            if o.Error == gorm.ErrRecordNotFound {
                continue
            }

            if !database.HandleLastError(o, c) {
                return
            }

            if client.Nume == "" {
                client.Nume = result.Nume
            }
            if client.Prenume == "" {
                client.Prenume = result.Prenume
            }
            break
        }
    }

    setMask := getClientMask(&client)
    if setMask.Count() != 2 {
        err := fmt.Errorf("default values for some fields not found in the other databases")
        c.Error(err)
        return
    }

    otherDbFields := alreadyExistsMask.Difference(setMask)
    pushClientData(tx, &client, otherDbFields)
    
    if !database.HandleLastError(tx, c) {
        return
    }
}

func InitHandler(builder *gin.Engine, appContext *stuff.ApplicationContext) {
    builder.POST("client", func(c *gin.Context) {
        errScope := stuff.CreateErrorScope(c)
        handle(c, appContext)
        if errScope.HasErrors() {
            return
        }
    })
}
