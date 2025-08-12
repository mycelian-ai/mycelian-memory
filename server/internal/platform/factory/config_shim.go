package factory

// ConfigShim is a minimal struct to satisfy NewStorage for tests without
// importing the full server/internal/config package.
type ConfigShim struct {
	DBDriver    string
	PostgresDSN string
}

// Expose required fields to NewStorage via getters to mimic config.Config.
func (c *ConfigShim) GetDBDriver() string    { return c.DBDriver }
func (c *ConfigShim) GetPostgresDSN() string { return c.PostgresDSN }
