package formhandler

import (
	"fmt"
	"webapp/database"
	"webapp/stuff"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"common/models"
	clientmod "common/models/client"
)


type ClientIdentificationDto struct {
    ID clientmod.IDType `form:"id"`
    Email string `form:"email"`
}

func getMask(client ClientIdentificationDto) clientmod.FieldMask {
    ret := clientmod.FieldMask{}
    if client.ID != 0 {
        ret.SetId(true)
    }
    if client.Email != "" {
        ret.SetEmail(true)
    }
    return ret
}

var errNoClient = fmt.Errorf("NoClientError")

func bindDto(c *gin.Context) (ret ClientIdentificationDto, err error) {
    err = c.BindQuery(&ret)
    if err != nil {
        return
    }

    mask := getMask(ret)
    if mask.Empty() {
        err = errNoClient
        return 
    } else if mask.Count() == 2 {
        err = fmt.Errorf("both id or email provided, must only provide one of the two")
    }

    if err != nil {
        validationErr := stuff.ValidationError(err)
        c.Error(validationErr)
    }
    return
}

func query(c *gin.Context, appContext *stuff.ApplicationContext) (ret database.Client) {

    identification, err := bindDto(c)
    if err != nil {
        return
    }

    errScope := stuff.CreateErrorScope(c)
    gormDb := appContext.GormFactory.CreateWrapError(c, appContext.DbContext.MainDatabaseIndex)
    if errScope.HasErrors() {
        return
    }

    q := gormDb
    q = q.Model(&database.Client{})
    if identification.Email != "" {
        q = q.Where("email = ?", identification.Email)
    } else {
        q = q.Where("id = ?", identification.ID)
    }

    q = q.First(&ret)

    if q.Error != gorm.ErrRecordNotFound {
        database.HandleLastError(q, c)
    }
    if errScope.HasErrors() {
        return
    }
    return
}

func Handle(c *gin.Context, appContext *stuff.ApplicationContext) (ret models.Client) {
    errScope := stuff.CreateErrorScope(c)
    client := query(c, appContext)
    if errScope.HasErrors() {
        return
    }

    ret = client.ToDomainModel()
    return
}
