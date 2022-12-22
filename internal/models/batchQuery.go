package models

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/onlysumitg/qsql2/internal/batch"
	bolt "go.etcd.io/bbolt"
)

type BatchSql struct {
	Server
	RunningSql
	QueryResults []QueryResult
	Status       string
	Notified     bool
	CreatedAt    time.Time `json:"created_at" db:"created_at" form:"-"`
	CompletedAt  time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// Define a new UserModel type which wraps a database connection pool.
type BatchSQLModel struct {
	DB *bolt.DB
}

func (m *BatchSQLModel) getTableName() []byte {
	return []byte("batchsqls3")
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSQLModel) BatchProcess() *batch.BatchProcess {
	return &batch.BatchProcess{
		BatchFunction: m.ProcessPending,
		RunEvery:      time.Second * 30,
	}
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSQLModel) ProcessPending(mutex *sync.Mutex) {
	log.Println("Starting process pending Batch SQLs")
	mutex.Lock()
	defer mutex.Unlock()

	var wg sync.WaitGroup

	for _, batchSQL := range m.List() {
		if strings.EqualFold(batchSQL.Status, "PENDING") {
			wg.Add(1)
			go m.ProcessSingle(batchSQL, &wg)

		}
	}

	wg.Wait()

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (m *BatchSQLModel) ProcessSingle(u *BatchSql, wg *sync.WaitGroup) {
	defer wg.Done()

	u.Status = "STARTED"
	m.Update(u)

	u.QueryResults = make([]QueryResult, 0)

	PrepareSQLToRun(&u.RunningSql)
	queryResults := ActuallyRunSQL2(u.Server, u.RunningSql)
	
	for _, qr := range queryResults {
		log.Println("qr>>>", qr)
		u.QueryResults = append(u.QueryResults, *qr)
	}

	u.CompletedAt = time.Now()
	u.Status = "COMPLETED"
	m.Update(u)

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) Insert(u *BatchSql) (string, error) {
	var id string = uuid.NewString()
	u.RunningSql.ID = id
	u.Status = "PENDING"
	u.CreatedAt = time.Now()
	err := m.Update(u)

	return id, err
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Insert method to add a new record to the "users" table.
func (m *BatchSQLModel) Update(u *BatchSql) error {
	err := m.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(m.getTableName())
		if err != nil {
			return err
		}

		buf, err := json.Marshal(u)
		if err != nil {
			return err
		}

		// key = > user.name+ user.id
		key := strings.ToUpper(u.RunningSql.ID) // + string(itob(u.ID))

		return bucket.Put([]byte(key), buf)
	})

	return err

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *BatchSQLModel) List() []*BatchSql {
	batchSqls := make([]*BatchSql, 0)
	_ = m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {

			batchSql := BatchSql{}
			err := json.Unmarshal(v, &batchSql)
			if err == nil {
				batchSqls = append(batchSqls, &batchSql)
			}
		}

		return nil
	})
	return batchSqls

}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
// We'll use the Exists method to check if a user exists with a specific ID.
func (m *BatchSQLModel) Get(id string) (*BatchSql, error) {

	if id == "" {
		return nil, errors.New("blank id not allowed")
	}
	var serverJSON []byte // = make([]byte, 0)

	err := m.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(m.getTableName())
		if bucket == nil {
			return errors.New("table does not exits")
		}
		serverJSON = bucket.Get([]byte(strings.ToUpper(id)))

		return nil

	})
	batchSql := BatchSql{}
	if err != nil {
		return &batchSql, err
	}

	// log.Println("serverJSON >2 >>", serverJSON)

	if serverJSON != nil {
		err := json.Unmarshal(serverJSON, &batchSql)
		return &batchSql, err

	}

	return &batchSql, ErrServerNotFound

}
