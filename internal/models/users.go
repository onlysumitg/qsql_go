package models

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new User type. Notice how the field names and types align
// with the columns in the database "users" table?
type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
	ResetPassword  bool
}

func (u *User) setEncruptedPassword(password string) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		u.HashedPassword = []byte(password)
	} else {
		u.HashedPassword = hash
	}

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new UserModel type which wraps a database connection pool.
type UserModel struct {
	DB *bolt.DB
}

func (m *UserModel) getTableName() []byte {
	return []byte("users")
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
type UserModelInterface interface {
	Insert(name, email, password string) error
	Authenticate(email, password string) (int, error)
	Exists(id int) (bool, error)
	Get(id int) (*User, error)
	PasswordUpdate(id int, currentPassword, newPassword string) error
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *UserModel) Insert(u *User) error {

	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}

		// Generate ID for the user.
		// This returns an error only if the Tx is closed or not writeable.
		// That can't happen in an Update() call so I ignore the error check.
		id, _ := bucket.NextSequence()
		u.ID = int(id)
		// Marshal user data into bytes.
		buf, err := json.Marshal(u)
		if err != nil {
			return err
		}

		// key = > user.name+ user.id
		key := strings.ToUpper(u.Name) // + string(itob(u.ID))

		return bucket.Put([]byte(key), buf)
	})

	if err != nil {
		log.Fatal(err)
	}

	return nil
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Authenticate method to verify whether a user exists with
// the provided email address and password. This will return the relevant
// user ID if they do.
func (m *UserModel) Authenticate(name, password string) *User {
	user, err := m.Get(name)
	if err != nil {
		return &User{}
	}

	err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(password))
	if err != nil {
		return &User{}
	}

	return user
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *UserModel) Exists(name string) bool {

	var userJson []byte

	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		key := strings.ToUpper(name)

		userJson = bucket.Get([]byte(key))

		return nil

	})

	return (userJson != nil)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *UserModel) Get(name string) (*User, error) {

	var userJson []byte // = make([]byte, 0)

	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		key := strings.ToUpper(name)

		userJson = bucket.Get([]byte(key))

		return nil

	})

	user := User{}

	if userJson != nil {
		err := json.Unmarshal(userJson, &user)
		if err != nil {
			return &user, nil
		}
	}

	return nil, ErrUserNotFound

}
