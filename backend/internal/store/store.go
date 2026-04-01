package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("username or email already taken")
	usersBucket     = []byte("users")
	emailIndex      = []byte("email_index")
)

// User represents a registered user.
type User struct {
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

// Store wraps BoltDB for user persistence.
type Store struct {
	db *bolt.DB
}

// New opens or creates the BoltDB file at dbPath.
func New(dbPath string) (*Store, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(usersBucket); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(emailIndex)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create buckets: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// CreateUser stores a new user, returning ErrUserExists if the username or email is taken.
func (s *Store) CreateUser(user *User) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		ub := tx.Bucket(usersBucket)
		eb := tx.Bucket(emailIndex)
		if ub.Get([]byte(user.Username)) != nil {
			return ErrUserExists
		}
		if eb.Get([]byte(user.Email)) != nil {
			return ErrUserExists
		}
		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		if err := ub.Put([]byte(user.Username), data); err != nil {
			return err
		}
		return eb.Put([]byte(user.Email), []byte(user.Username))
	})
}

// GetUser looks up a user by username.
func (s *Store) GetUser(username string) (*User, error) {
	var user User
	err := s.db.View(func(tx *bolt.Tx) error {
		data := tx.Bucket(usersBucket).Get([]byte(username))
		if data == nil {
			return ErrUserNotFound
		}
		return json.Unmarshal(data, &user)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}
