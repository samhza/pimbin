package pimbin

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3" // because we need the driver
)

const schema = `
CREATE TABLE users (
	username VARCHAR(255) PRIMARY KEY,
	password VARCHAR(255) NOT NULL,
    token    CHAR(24) UNIQUE
);
CREATE TABLE pastes (
	id     CHAR(6) PRIMARY KEY NOT NULL,
	owner  VARCHAR(255) NOT NULL,
	FOREIGN KEY(owner) REFERENCES users(username)
);
CREATE TABLE files (
	paste CHAR(6) NOT NULL,
	hash  CHAR(44) NOT NULL,
	name  VARCHAR(128) NOT NULL,
	FOREIGN KEY(paste) REFERENCES pastes(id) ON DELETE CASCADE
);`

var migrations = []string{""}

// User contains a user's data.
type User struct {
	Password string
	Name     string
	Token    string
}

// Paste contains a paste's data.
type Paste struct {
	ID    string
	Owner string
	Files []File
}

// File describes a paste's file.
type File struct {
	Hash string
	Name string
}

// DB is a pimbin database.
type DB struct {
	lock sync.RWMutex
	db   *sql.DB
}

// OpenSQLiteDB opens and returns an sqlite3 database from the path provided.
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

func fromStringPtr(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func toStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Users lists the users in the database.
func (db *DB) Users() ([]User, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	rows, err := db.db.Query("SELECT username, password, token FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var (
			user     User
			password *string
			token    *string
		)
		if err := rows.Scan(&user.Name, &password, &token); err != nil {
			return nil, err
		}
		user.Password = fromStringPtr(password)
		user.Token = fromStringPtr(token)
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// User returns a user from their username, or an error if one occurs while trying to retrieve them.
func (db *DB) User(username string) (*User, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	user := &User{Name: username}

	var password *string
	row := db.db.QueryRow("SELECT password FROM users WHERE username = ?", username)
	if err := row.Scan(&password); err != nil {
		return nil, err
	}
	user.Password = fromStringPtr(password)
	return user, nil
}

// CreateUser inserts user into the database.
func (db *DB) CreateUser(user *User) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	password := toStringPtr(user.Password)
	_, err := db.db.Exec("INSERT INTO users(username, password) VALUES (?, ?)", user.Name, password)
	return err
}

// RefreshToken refreshes a user's token and returns the new token.
func (db *DB) RefreshToken(user *User) (string, error) {
	db.lock.Lock()
	defer db.lock.Unlock()

	b := make([]byte, 24)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)
	_, err = db.db.Exec("UPDATE users SET token = ? WHERE username = ?", token, user.Name)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return db.RefreshToken(user)
	}
	return token, err
}

// UpdatePassword records changes to user's password in the database.
func (db *DB) UpdatePassword(user *User) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	password := toStringPtr(user.Password)
	_, err := db.db.Exec(`UPDATE users SET password = ? WHERE username = ?`,
		password, user.Name)
	return err
}

// PutPaste inserts the given paste into the database.
func (db *DB) PutPaste(p Paste) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	_, err := db.db.Exec("INSERT INTO pastes(id,owner) VALUES(?, ?)", p.ID, p.Owner)
	if err != nil {
		return err
	}
	stmt, err := db.db.Prepare("INSERT INTO files(paste, hash, name) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, f := range p.Files {
		if _, err = stmt.Exec(p.ID, f.Hash, f.Name); err != nil {
			return err
		}
	}
	return nil
}

// Paste returns a paste from its ID.
func (db *DB) Paste(id string) (*Paste, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var owner string
	row := db.db.QueryRow("SELECT owner FROM pastes WHERE id=?", id)
	err := row.Scan(&owner)
	if err != nil {
		return nil, err
	}

	rows, err := db.db.Query("SELECT hash,name FROM files WHERE paste=?", id)
	if err != nil {
		return nil, err
	}
	paste := &Paste{
		ID:    id,
		Owner: owner,
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
		paste.Files = append(paste.Files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return paste, nil
}

// DeletePaste deletes a paste and its file entries by the id.
func (db *DB) DeletePaste(id string) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	_, err := db.db.Exec("DELETE FROM paste WHERE id = ?", id)
	if err != nil {
		return err
	}
	return nil
}

// Close closes the DB.
func (db *DB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()
	return db.Close()
}
