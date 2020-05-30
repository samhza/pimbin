package main

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const schema = `
CREATE TABLE User (
	username VARCHAR(255) PRIMARY KEY,
	password VARCHAR(255) NOT NULL
);
CREATE TABLE Paste (
	owner TEXT,
	id   TEXT,
    hash TEXT,
    name TEXT
);
`

var migrations = []string{""}

type User struct {
	Password string
	Username string
	Token    string
}

type Paste struct {
	ID    string
	Owner string
	Files map[string]string
}

type File struct {
	Hash string
	Name string
}

type DB struct {
	lock sync.Mutex
	db   *sql.DB
}

func OpenSQLiteDB(source string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", source)
	if err != nil {
		return nil, err
	}

	db := &DB{db: sqlDB}
	if err := db.migrate(); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) migrate() error {
	db.lock.Lock()
	defer db.lock.Unlock()
	tx, err := db.db.Begin()
	defer tx.Rollback()
	if err != nil {
		return fmt.Errorf("couldn't start db transaction", err)
	}
	var version int
	// var ver int
	if err := db.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("couldn't query schema version: %v", err)
	}
	if version > len(migrations) {
		log.Fatalln("database is from a newer pimbin")
	}
	if version == 0 {
		if _, err := tx.Exec(schema); err != nil {
			return fmt.Errorf("failed while executing schema: %v", err)
		}
		version++
	}
	for version < len(migrations) {
		tx.Exec(migrations[version])
		version++
	}
	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", len(migrations)))
	if err != nil {
		return fmt.Errorf("failed to change schema version: %v", err)
	}
	return tx.Commit()
}

func (db *DB) GetUser(username string) (*User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	user := &User{Username: username}

	var password string
	row := db.db.QueryRow("SELECT password FROM User WHERE username = ?", username)
	if err := row.Scan(&password); err != nil {
		return nil, err
	}
	user.Password = password
	return user, nil
}

func (db *DB) CreateUser(user *User) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	password := user.Password
	_, err := db.db.Exec("INSERT INTO User(username, password) VALUES (?, ?)", user.Username, password)
	return err
}

func (db *DB) PutPaste(p Paste) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	var err error
	for k, v := range p.Files {
		_, err = db.db.Exec("INSERT INTO Paste(owner, id, hash, name) VALUES (?, ?, ?, ?)", "name", p.ID, v, k)
		if err != nil {
			break
		}
	}
	return err
}

func (db *DB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()
	return db.Close()
}
