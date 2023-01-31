package models

import (
	"encoding/json"
	"errors"
	"log"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/validator"
	bolt "go.etcd.io/bbolt"
)

// -----------------------------------------------------------------
// -----------------------------------------------------------------
// Define a new User type. Notice how the field names and types align
// with the columns in the database "users" table?
type SavedQuery struct {
	ID                  string        `json:"id" db:"id" form:"id"`
	Name                string        `json:"name" db:"name" form:"name"`
	Category            string        `json:"category" db:"category" form:"category"`
	Sql                 string        `json:"sql" db:"sql" form:"sql"`
	Fields              []*QueryField `json:"fields" db:"-" form:"-"`
	validator.Validator               // this contains the fielderror
}

type SavedQueryBuild struct {
	SqlToRun    string            `json:"sqltorun" db:"-" form:"-"`
	FieldErrors map[string]string `json:"fielderrors" db:"-" form:"-"`
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s *SavedQuery) PopulateFields() {
	s.Fields = findQueryFields(s.Sql)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (s *SavedQuery) ReplaceFields(values map[string]string) (string, map[string]string) {

	return ReplaceQueryFields(s.Sql, values)
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func ReplaceQueryFields(sqlString string, values map[string]string) (string, map[string]string) {

	fieldErrors := make(map[string]string)
	fields := findQueryFields(sqlString)
	sql := sqlString
	log.Println("sql1>>>>>", sql, fields)

	for _, field := range fields {
		fieldValue, found := values[field.Name]
		if found {
			sql = strings.ReplaceAll(sql, field.ID, fieldValue)
			log.Println("sql>>>>>", sql)
		} else if field.DefaultValue != "" {
			sql = strings.ReplaceAll(sql, field.ID, field.DefaultValue)
		} else {
			fieldErrors[field.Name] = "Field value is required"
		}

	}
	return sql, fieldErrors
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func findQueryFields(str string) []*QueryField {

	var re = regexp.MustCompile(`(?m)({{.*?}})`)

	fields := make([]*QueryField, 0)
	fieldNames := make([]string, 0)

	for _, match := range re.FindAllString(str, -1) {
		field := fieldToQueryField(match)

		if !isInList(fieldNames, field.Name) { // not found
			fieldNames = append(fieldNames, field.Name)
			fields = append(fields, field)
		}

		//fmt.Println(match, "found at index", i)
	}
	return fields
}

// -----------------------------------------------------------------
//
//	TODO --> improve search
//
// -----------------------------------------------------------------
func isInList(list []string, search string) bool {
	for _, val := range list {
		if strings.EqualFold(val, search) {
			return true
		}
	}

	return false
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------

func fieldToQueryField(str string) *QueryField {
	field := strings.TrimRight(str, "}")
	field = strings.TrimLeft(field, "{")

	fieldNameValue := strings.Split(field, ":")

	queryField := QueryField{ID: str}
	queryField.Name = strings.Trim(fieldNameValue[0], " ")

	if len(fieldNameValue) > 1 {
		queryField.DefaultValue = fieldNameValue[1]
	}

	return &queryField
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new UserModel type which wraps a database connection pool.
type SavedQueryModel struct {
	DB *bolt.DB
}

func (m *SavedQueryModel) getTableName() []byte {
	return []byte("savedquery")
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *SavedQueryModel) Save(u *SavedQuery) (string, error) {
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
		u.Category = strings.ToUpper(strings.TrimSpace(u.Category))
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

	go AutoGenerateForSavedQueries(m)

	return id, err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *SavedQueryModel) Delete(id string) error {

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
func (m *SavedQueryModel) Exists(id string) bool {

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
func (m *SavedQueryModel) DuplicateName(name string) bool {
	exists := false
	for _, savedQuery := range m.List() {

		if strings.EqualFold(savedQuery.Name, name) {
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
func (m *SavedQueryModel) Get(id string) (*SavedQuery, error) {

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
	savedQuery := SavedQuery{}
	if err != nil {
		return &savedQuery, err
	}

	// log.Println("savedQueryJSON >2 >>", savedQueryJSON)

	if savedQueryJSON != nil {
		err := json.Unmarshal(savedQueryJSON, &savedQuery)
		return &savedQuery, err

	}

	return &savedQuery, ErrSavedQueryNotFound

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *SavedQueryModel) List() []*SavedQuery {
	savedQueries := make([]*SavedQuery, 0)
	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {

			savedQuery := SavedQuery{}
			err := json.Unmarshal(v, &savedQuery)
			if err == nil {
				savedQueries = append(savedQueries, &savedQuery)
			}
		}

		return nil
	})
	return savedQueries

}
