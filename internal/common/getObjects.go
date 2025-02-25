package common

// Entry is a struct holding the json-ld metadata and data (the text)
type Entry struct {
	Bucketname string
	Key        string
	Urlval     string
	Sha1val    string
	Jld        string
}
