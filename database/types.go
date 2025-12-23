package database

// TODO: Should this become a interface instead?
type Entry struct {
	Key       string
	Value     string
	Tombstone bool
}

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
