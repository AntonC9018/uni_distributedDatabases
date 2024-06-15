package stuff

import (
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/go-ozzo/ozzo-validation"
	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
)

type Cursor = paginator.Cursor

type valueCursor struct {
    // For some reason, query parameters are called "form" in gin
	After  string `form:"after"`
	Before string `form:"before"`
}

type Pagination struct {
    cursor valueCursor
    Order paginator.Order
    Limit int
}

func (p *Pagination) Cursor() Cursor {
    ret := Cursor{
        After: &p.cursor.After,
        Before: &p.cursor.Before,
    }
    if *ret.After == "" {
        ret.After = nil
    }
    if *ret.Before == "" {
        ret.Before  = nil
    }
    return ret
}

func GetPagination(c *gin.Context) Pagination {
    errScope := CreateErrorScope(c)
    ret := Pagination{}

    err := c.BindQuery(&ret.cursor)
    if err != nil {
        c.Error(err)
    }

    err = c.BindQuery(&ret.Order)
    if err != nil {
        c.Error(err)
    }

    err = c.BindQuery(&ret.Limit)
    if err != nil {
        c.Error(err)
    }

    if errScope.HasErrors() {
        return ret
    }

    {
        err := validation.ValidateStruct(&ret,
            validation.Field(
                &ret.Limit,
                validation.Max(100),
                validation.Min(20)),
            validation.Field(
                &ret.Order,
                validation.In(paginator.ASC, paginator.DESC)))
        if err != nil {
            c.Error(err)
        }
    }

    if ret.Limit == 0 {
        ret.Limit = 100
    }
    if ret.Order == "" {
        ret.Order = paginator.ASC
    }

    return ret
}

func CreatePaginator(c *gin.Context) paginator.Paginator {
    ret := paginator.Paginator{}

    errScope := CreateErrorScope(c)
    p := GetPagination(c)
    if errScope.HasErrors() {
        return ret
    }

    ret.SetAllowTupleCmp(false)
    ret.SetKeys("ID")
    ret.SetOrder(p.Order)
    ret.SetLimit(p.Limit)

    {
        temp := p.cursor.After
        if temp != "" {
            ret.SetAfterCursor(temp)
        }
    }
    {
        temp := p.cursor.Before
        if temp != "" {
            ret.SetBeforeCursor(temp)
        }
    }
    return ret
}

func ReplaceCursorInQuery(url *url.URL, cursor Cursor) {
    queryParams := url.Query()
    replace := func(key string, value *string) {
        if value == nil {
            queryParams.Del(key)
            return
        }
        v, exists := queryParams[key]
        if exists {
            v[0] = *value
        } else {
            queryParams[key] = []string{*value}
        }
    }
    replace("before", cursor.Before)
    replace("after", cursor.After)

    url.RawQuery = queryParams.Encode()
}
