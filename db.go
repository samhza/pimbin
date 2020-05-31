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
	Files []File
}

type File struct {
	Hash string
	Name string
}

type DB struct {
	lock sync.RWMutex
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
		return fmt.Errorf("couldn't start db transaction: %v", err)
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
	stmt, err := db.db.Prepare("INSERT INTO Paste(owner, id, hash, name) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, f := range p.Files {
		if _, err = stmt.Exec("name", p.ID, f.Hash, f.Name); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) GetPaste(id string) (*Paste, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()
	rows, err := db.db.Query("SELECT hash,name FROM Paste WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	paste := &Paste{
		ID: id,
	}
	for rows.Next() {
		var (
			hash string
			name string
		)
		if err := rows.Scan(&hash, &name); err != nil {
			return nil, err
		}
		file := File{
			Hash: hash,
			Name: name,
		}
		println(hash, name)
		paste.Files = append(paste.Files, file)
	}
	return paste, nil
}

func (db *DB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()
	return db.Close()
}
