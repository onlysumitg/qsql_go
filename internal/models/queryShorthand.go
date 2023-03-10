package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/validator"
	bolt "go.etcd.io/bbolt"
)

func buildQueryAliasWithParamters(sql string, params []string) (string, string) {

	queryFields := findQueryFields(sql)
	if len(queryFields) == 0 {
		log.Println(">>>>> queryFields element count", len(queryFields), sql)
		return sql, ""
	}

	values := make(map[string]string)

	for i, queryField := range queryFields {
		if len(params) > i {
			values[queryField.Name] = params[i]
		}
	}

	finalQuery, fieldErrors := ReplaceQueryFields(sql, values)

	finalErrorString := ""
	seperator := ""
	for key, value := range fieldErrors {
		finalErrorString = fmt.Sprintf("%s%s%s:%s", finalErrorString, seperator, key, value)
		seperator = " | "
	}

	return finalQuery, finalErrorString
}

// -----------------------------------------------------------------
// -----------------------------------------------------------------
// Define a new User type. Notice how the field names and types align
// with the columns in the database "users" table?
type ShorthandQuery struct {
	ID                  string `json:"id" db:"id" form:"id"`
	Name                string `json:"name" db:"name" form:"name"`
	Sql                 string `json:"sql" db:"sql" form:"sql"`
	validator.Validator        // this contains the fielderror
	AutoGenerated       bool   `json:"autogen" db:"autogen" form:"autogen"`
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new UserModel type which wraps a database connection pool.
type ShorthandQueryModel struct {
	DB *bolt.DB
}

func (m *ShorthandQueryModel) getTableName() []byte {
	return []byte("queryalias")
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------

func AutoGenerateForSavedQueries(savedQueries *SavedQueryModel) {

	shorthandQueriesModel := ShorthandQueryModel{DB: savedQueries.DB}

	shorthandQueriesModel.ClearAutoGenerated()

	for _, savedQuery := range savedQueries.List() {
		key := fmt.Sprintf("@%s_%s", savedQuery.Category, savedQuery.Name)
		key = strings.TrimSpace(key)
		key = strings.ReplaceAll(key, " ", "_")
		value := savedQuery.Sql

		// skip multiple queries
		if strings.Contains(value, ";") {
			continue
		}

		//if user has saved a alias with same key --> dont replace it
		if !shorthandQueriesModel.NameExists(key) {
			shorthandQuery := ShorthandQuery{Name: key, Sql: value, AutoGenerated: true}
			shorthandQueriesModel.Save(&shorthandQuery)

		}

	}

	ReloadQueryMap(&shorthandQueriesModel)

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *ShorthandQueryModel) Save(u *ShorthandQuery) (string, error) {
	var id string
	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		u.Name = strings.ToUpper(strings.TrimSpace(u.Name))

		// generate new ID if id is blank else use the old one to update
		if u.ID == "" {
			u.ID = uuid.NewString()
		}
		id = u.ID
		// Marshal user data into bytes.
		buf, err := json.Marshal(u)
		if err != nil {
			return err
		}

		// key = > user.name+ user.id
		key := strings.ToUpper(u.ID) // + string(itob(u.ID))

		return bucket.Put([]byte(key), buf)
	})
	 

	return id, err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *ShorthandQueryModel) Delete(id string) error {

	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		key := strings.ToUpper(id)
		dbDeleteError := bucket.Delete([]byte(key))
		return dbDeleteError
	})

	return err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ShorthandQueryModel) Exists(id string) bool {

	var userJson []byte

	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}
		key := strings.ToUpper(id)

		userJson = bucket.Get([]byte(key))

		return nil

	})

	return (userJson != nil)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ShorthandQueryModel) NameExists(name string) bool {

	exists := false
	for _, shorthandQuery := range m.List() {

		if strings.EqualFold(shorthandQuery.Name, strings.ToUpper(strings.TrimSpace(name))) {
			exists = true
			break
		}
	}

	return exists
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ShorthandQueryModel) DuplicateName(u *ShorthandQuery) bool {
	exists := false
	for _, shorthandQuery := range m.List() {

		if strings.EqualFold(shorthandQuery.Name, u.Name) && !strings.EqualFold(shorthandQuery.ID, u.ID) {
			exists = true
			break
		}
	}

	return exists
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ShorthandQueryModel) Get(id string) (*ShorthandQuery, error) {

	if id == "" {
		return nil, errors.New("SavedQuery blank id not allowed")
	}
	var savedQueryJSON []byte // = make([]byte, 0)

	err := m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		savedQueryJSON = bucket.Get([]byte(strings.ToUpper(id)))

		return nil

	})
	shorthandQuery := ShorthandQuery{}
	if err != nil {
		return &shorthandQuery, err
	}

	// log.Println("savedQueryJSON >2 >>", savedQueryJSON)

	if savedQueryJSON != nil {
		err := json.Unmarshal(savedQueryJSON, &shorthandQuery)
		return &shorthandQuery, err

	}

	return &shorthandQuery, ErrSavedQueryNotFound

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ShorthandQueryModel) List() []*ShorthandQuery {
	savedQueries := make([]*ShorthandQuery, 0)
	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {

			shorthandQuery := ShorthandQuery{}
			err := json.Unmarshal(v, &shorthandQuery)
			if err == nil {
				savedQueries = append(savedQueries, &shorthandQuery)
			}
		}

		return nil
	})
	return savedQueries

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *ShorthandQueryModel) ClearAutoGenerated() {
	shortHandQueries := m.List()

	for _, shortHandQuery := range shortHandQueries {
		if shortHandQuery.AutoGenerated {
			m.Delete(shortHandQuery.ID)
		}
	}

}
