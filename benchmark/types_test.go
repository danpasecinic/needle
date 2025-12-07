package benchmark

type Config struct {
	Host string
	Port int
}

type Logger struct {
	Level string
}

type Database struct {
	Config *Config
	Logger *Logger
}

type Cache struct {
	Logger *Logger
}

type Repository struct {
	DB    *Database
	Cache *Cache
}

type Service struct {
	Repo   *Repository
	Logger *Logger
}
