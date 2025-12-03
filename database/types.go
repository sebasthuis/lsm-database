package database

type LookupStatus int

const (
	NotFound LookupStatus = iota
	Found
	Deleted
)

type LookupResult struct {
	Value  string
	Status LookupStatus
}
