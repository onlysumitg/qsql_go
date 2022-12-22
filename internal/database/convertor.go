package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zerobit-tech/godbc/database/sql"
)

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func prepareColumnType(sql_column sql.ColumnType) (column_type ColumnType) {

	name := sql_column.Name()
	length, hasLength := sql_column.Length()
	precision, scale, hasPrecisionScale := sql_column.DecimalSize()
	nullable, hasNullable := sql_column.Nullable()
	databaseType := sql_column.DatabaseTypeName()

	column_type = ColumnType{Name: name,
		Length:            length,
		HasLength:         hasLength,
		Precision:         precision,
		Scale:             scale,
		HasPrecisionScale: hasPrecisionScale,
		Nullable:          nullable,
		HasNullable:       hasNullable,
		DatabaseType:      databaseType,
	}
	return
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func prepareColumnTypes(rows sql.Rows, wg *sync.WaitGroup, colch chan<- []ColumnType) { // (column_types []ColumnType) {
	column_types_p, _ := rows.ColumnTypes()

	column_types := make([]ColumnType, 0)
	for _, sql_column := range column_types_p {
		column_types = append(column_types, prepareColumnType(*sql_column))
	}
	wg.Done()
	colch <- column_types
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func processRow(scans []interface{}, fields []string) map[string]interface{} {
	row := make(map[string]interface{})
	for i, v := range scans {

		// fmt.Println(">>>>>>>", fields[i], " type = ", reflect.TypeOf(v), reflect.ValueOf(v).Kind())

		switch v.(type) {
		case []uint, []uint8:

			row[fields[i]] = fmt.Sprintf("%s", v)
		default:
			row[fields[i]] = v
		}
		// if reflect.TypeOf(v) == []byte {
		// 	row[fields[i]] = string(v)
		// }
		// else {
		// row[fields[i]] = v
		// }

	}
	return row
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func ToMap(rows *sql.Rows, maxRows int, scrollTo int) (return_rows []map[string]interface{}, column_types []ColumnType) {
	fields, _ := rows.Columns()
	log.Println("p1.3.1", time.Now())

	colch := make(chan []ColumnType)
	defer close(colch)

	var wg sync.WaitGroup
	wg.Add(1)
	go prepareColumnTypes(*rows, &wg, colch)

	log.Println("p1.3.2", time.Now())

	fmt.Println("rows.JumpToRow2(3)", rows.JumpToRow2(scrollTo))
	for rows.Next() {
		//rows.JumpToRow(3)

		log.Println("p1.3.2.1", time.Now())

		scans := make([]interface{}, len(fields))

		for i := range scans {
			scans[i] = &scans[i]
		}

		rows.Scan(scans...)

		log.Println("p1.3.2.2", time.Now())

		return_rows = append(return_rows, processRow(scans, fields))
		log.Println("p1.3.2.3", time.Now())
		if maxRows > 0 && len(return_rows) >= maxRows {
			break
		}
	}

	log.Println("p1.3.3", time.Now())
	wg.Wait()

	column_types = <-colch
	log.Println("p1.3.4", time.Now())

	return
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func processRow2(scans []interface{}, fields []string, rowch chan<- map[string]interface{}, wg *sync.WaitGroup) { //map[string]interface{} {
	wg.Add(1)
	row := make(map[string]interface{})
	for i, v := range scans {

		// fmt.Println(">>>>>>>", fields[i], " type = ", reflect.TypeOf(v), reflect.ValueOf(v).Kind())

		switch v.(type) {
		case []uint, []uint8:

			row[fields[i]] = fmt.Sprintf("%s", v)
		default:
			row[fields[i]] = v
		}
		// if reflect.TypeOf(v) == []byte {
		// 	row[fields[i]] = string(v)
		// }
		// else {
		// row[fields[i]] = v
		// }

	}
	wg.Done()
	rowch <- row
}

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func ToMap2(rows *sql.Rows, maxRows int, scrollTo int) (return_rows []map[string]interface{}, column_types []ColumnType) {
	fields, _ := rows.Columns()
	log.Println("p1.3.1", time.Now())

	colch := make(chan []ColumnType)
	defer close(colch)

	var wg sync.WaitGroup
	wg.Add(1)
	go prepareColumnTypes(*rows, &wg, colch)

	log.Println("p1.3.2", time.Now())
	rowch := make(chan map[string]interface{})
	defer close(rowch)

	rowCount := 0

	fmt.Println("rows.JumpToRow2(3)", rows.JumpToRow2(scrollTo))

	for rows.Next() {
		//rows.JumpToRow(3)

		log.Println("p1.3.2.1", time.Now())

		rowCount += 1
		scans := make([]interface{}, len(fields))

		for i := range scans {
			scans[i] = &scans[i]
		}

		rows.Scan(scans...)

		log.Println("p1.3.2.2", time.Now())

		go processRow2(scans, fields, rowch, &wg)
		log.Println("p1.3.2.3", time.Now())
		if rowCount >= maxRows {
			break
		}
	}

	log.Println("p1.3.3", time.Now())
	wg.Wait()

	column_types = <-colch
	log.Println("p1.3.4", time.Now())

	for i := 1; i <= rowCount; i++ {
		row := <-rowch

		return_rows = append(return_rows, row)

	}

	log.Println("p1.3.5", time.Now())

	return
}
