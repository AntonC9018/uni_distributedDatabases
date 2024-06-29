package createhandler

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
	"webapp/features/clients/cookies"
)

// This does:
// Create if doesn't exist;
// Update if given new names;
// If not given new names, the client is looked up in the database;
// If there's no record with this email in the database, it's looked up in the other databases.
//
// I know it doesn't make much sense for a real app,
// but it does make sense for testing without real authentication.
func Handle(c *gin.Context, appContext *stuff.ApplicationContext) {
    errScope := stuff.CreateErrorScope(c)
    dto := bindDtoWithValidation(c)
    if errScope.HasErrors() {
        return
    }

    // First look in the main database
    dbIndex := appContext.DbContext.MainDatabaseIndex
    gormDb := appContext.GormFactory.CreateWrapError(c, dbIndex)
    if errScope.HasErrors() {
        return
    }

    client := mapToDbObject(dto)

    tx := gormDb.Begin()
    defer func(){
        if errScope.HasErrors() {
            tx.Rollback()
        } else {
            tx.Commit()
        }
    }()

    q := tx.Model(&database.Client{})
    q = q.Where("email = ?", client.Email)


    alreadyExistsMask := getClientMask(&client)
    const updatedFieldCount = 2

    // Nothing provided (neither name)
    if alreadyExistsMask.Empty() {
        // Maybe get the client data from the database.
        q = q.First(&client)

        isNotFound := q.Error == gorm.ErrRecordNotFound
        if !isNotFound {
            database.HandleLastError(q, c)
        }

    } else {
        // Try and update/create with the data that's been provided.
        q = pushClientData(q, &client, alreadyExistsMask)
        database.HandleLastError(q, c)
    }

    if errScope.HasErrors() {
        return
    }

    defer func() {
        if errScope.HasErrors() {
            return
        }

        if dto.Remember {
            stuff.SetCookie(c, &cookies.EmailCookie, client.Email)
        } else {
            stuff.RemoveCookie(c, &cookies.EmailCookie)
        }
    }()
    // At this point we've pulled in data from the db.
    // If it's just been created, and not all fields have been provided,
    // the fields that have not been filled in will be empty strings.
    alreadyExistsMask = getClientMask(&client)
    if alreadyExistsMask.Count() == updatedFieldCount {
        return
    }

    // Find the values for the other fields in the other databases.
    {
        dbs := appContext.DbContext.Databases

        // TODO: Multithread?
        for i := range dbs {
            if i == dbIndex {
                continue
            }

            otherGormDb := appContext.GormFactory.CreateWrapError(c, i)
            if otherGormDb == nil {
                continue
            }

            tempClient := database.Client{}

            o := otherGormDb
            o = o.Model(&database.Client{})
            o = o.Where("email = ?", dto.Email)
            o = o.First(&tempClient)

            if o.Error == gorm.ErrRecordNotFound {
                continue
            }

            if !database.HandleLastError(o, c) {
                return
            }

            if client.Nume == "" {
                client.Nume = tempClient.Nume
            }
            if client.Prenume == "" {
                client.Prenume = tempClient.Prenume
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
    q = tx
    q = pushClientData(q, &client, otherDbFields)
    
    if !database.HandleLastError(q, c) {
        return
    }
}

type CreateClientDto struct {
    Email string `form:"email"`
    Nume string `form:"nume"`
    Prenume string `form:"prenume"`
    Remember bool `form:"remember"`
}

func mapToDbObject(client CreateClientDto) database.Client {
    return database.Client{
        Email: client.Email,
        Nume: client.Nume,
        Prenume: client.Prenume,
    }
}

func getClientMask(client *database.Client) clientmod.FieldMask {
    empty := database.Client{}
    return getClientDifferenceMask(client, &empty)
}

func getClientDifferenceMask(a *database.Client, b *database.Client) clientmod.FieldMask {
    ret := clientmod.FieldMask{}
    if a.Nume != b.Nume {
        ret.SetNume(true)
    }
    if a.Prenume != b.Prenume {
        ret.SetPrenume(true)
    }
    return ret
}

func pushClientData(
    gormDb *gorm.DB,
    client *database.Client,
    maskToSet clientmod.FieldMask) *gorm.DB {

    updateColumnNames := make([]string, 0, maskToSet.Count())
    if maskToSet.Nume() {
        updateColumnNames = append(updateColumnNames, "nume")
    }
    if maskToSet.Prenume() {
        updateColumnNames = append(updateColumnNames, "prenume")
    }

    q := gormDb.Clauses(
        clause.OnConflict{
            Columns: []clause.Column{{ Name: "email" }},
            DoUpdates: clause.AssignmentColumns(updateColumnNames),
        },
        clause.Returning{})
    q = q.Create(client)
    return q
}

func bindDto(c *gin.Context) (ret CreateClientDto, err error) {
    err = c.Bind(&ret)
    if err != nil {
        c.Error(err)
    }
    return
}

func bindDtoWithValidation(c *gin.Context) (ret CreateClientDto) {
    ret, err := bindDto(c)
    if err != nil {
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

