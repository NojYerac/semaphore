package data

type DataSource interface{}

type Source struct {
	db DataSource
}

func NewSource(db DataSource) *Source {
	return &Source{
		db: db,
	}
}
