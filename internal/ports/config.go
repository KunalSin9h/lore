package ports

// ConfigPort reads and writes application configuration.
// Implemented by: adapters/sqlite.Config
type ConfigPort interface {
	Get(key string) (string, error)
	Set(key, value string) error
	All() (map[string]string, error)
}
