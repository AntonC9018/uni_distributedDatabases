package database

// Domain mapping
import (
	"common/models"
	"unsafe"
)

/*
type FoaieType string

type Foaie struct {
    ID int32
    Tip FoaieType
    Pret float64
    ProvidedTransport bool
    Hotel string
}

type Client struct {
    Id int32
    Email string
    Nume string
    Prenume string
}
*/

// The types I'm aliasing are actually domain models, not db models, but it's fine here
// because the fields are the same.


// For the purposes of strictness, I'm still going to do mappings.
type Foaie models.Foaie
func (m *Foaie) ToDomainModel() models.Foaie {
    return models.Foaie(*m)
}
func (m *Foaie) FromDomainModel(domainModel *models.Foaie) {
    *m = Foaie(*domainModel)
}
func ToDomailModelsFoaie(v []Foaie) []models.Foaie {
    return castSlice[Foaie, models.Foaie](v);
}

type Client models.Client
func (m *Client) ToDomainModel() models.Client {
    return models.Client(*m)
}
func (m *Client) FromDomainModel(domainModel *models.Client) {
    *m = Client(*domainModel)
}
func ToDomailModelsClient(v []Client) []models.Client {
    return castSlice[Client, models.Client](v);
}

func castSlice[From any, To any](v []From) []To {
    return unsafe.Slice((*To)(unsafe.Pointer(&v[0])), len(v))
}
