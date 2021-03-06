package ethutil

// Database interface
type Database interface {
	Put(key []byte, value []byte)
	Get(key []byte) ([]byte, error)
	LastKnownTD() []byte
	Close()
}
