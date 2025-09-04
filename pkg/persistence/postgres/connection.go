package postgres

import (
	"database/sql"
	"fmt"
	
	"github.com/elidor/dungeogo/pkg/persistence/interfaces"
	_ "github.com/lib/pq"
)

type PostgreSQLRepositoryManager struct {
	db               *sql.DB
	playerRepo       *PlayerRepository
	characterRepo    *CharacterRepository
	itemRepo         *ItemRepository
	worldRepo        *WorldRepository
}

func NewPostgreSQLRepositoryManager(databaseURL string) (*PostgreSQLRepositoryManager, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	manager := &PostgreSQLRepositoryManager{
		db: db,
	}
	
	manager.playerRepo = NewPlayerRepository(db)
	manager.characterRepo = NewCharacterRepository(db)
	manager.itemRepo = NewItemRepository(db)
	manager.worldRepo = NewWorldRepository(db)
	
	return manager, nil
}

func (m *PostgreSQLRepositoryManager) Players() interfaces.PlayerRepository {
	return m.playerRepo
}

func (m *PostgreSQLRepositoryManager) Characters() interfaces.CharacterRepository {
	return m.characterRepo
}

func (m *PostgreSQLRepositoryManager) Items() interfaces.ItemRepository {
	return m.itemRepo
}

func (m *PostgreSQLRepositoryManager) World() interfaces.WorldRepository {
	return m.worldRepo
}

func (m *PostgreSQLRepositoryManager) Close() error {
	return m.db.Close()
}

// GetDB returns the underlying database connection for testing
func (m *PostgreSQLRepositoryManager) GetDB() *sql.DB {
	return m.db
}