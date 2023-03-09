package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
)

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func prepareColumnType(sql_column sql.ColumnType, index int) (column_type ColumnType) {

	name := sql_column.Name()
	length, hasLength := sql_column.Length()
	precision, scale, hasPrecisionScale := sql_column.DecimalSize()
	nullable, hasNullable := sql_column.Nullable()
	databaseType := sql_column.DatabaseTypeName()

	column_type = ColumnType{
		IndexName:         fmt.Sprintf("%d_%s", index, name),
		Name:              name,
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
	for index, sql_column := range column_types_p {
		column_types = append(column_types, prepareColumnType(*sql_column, index))
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

		//fmt.Println(">>>>>>>", fields[i], " type = ", reflect.TypeOf(v), reflect.ValueOf(v).Kind())

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
	fieldsX, _ := rows.Columns()
	// fmt.Println("fields >>>", fieldsX)

	fields := make([]string, 0)
	for i, f := range fieldsX {
		fields = append(fields, fmt.Sprintf("%d_%s", i, f))
	}

	colch := make(chan []ColumnType)
	defer close(colch)

	var wg sync.WaitGroup
	wg.Add(1)
	go prepareColumnTypes(*rows, &wg, colch)

	//fmt.Println("ToMap rows.JumpToRow2(3)", rows.JumpToRow2(scrollTo))
	for rows.Next() {
		//rows.JumpToRow(3)

		scans := make([]interface{}, len(fields))

		for i := range scans {
			scans[i] = &scans[i]
		}

		err := rows.Scan(scans...)
		if err != nil {
			log.Println("ToMap Scan....:", err.Error())
		}

		return_rows = append(return_rows, processRow(scans, fields))
		if maxRows > 0 && len(return_rows) >= maxRows {
			break
		}
	}

	wg.Wait()

	column_types = <-colch

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
// NOT in USE
// -----------------------------------------------------------------
func ToMap2(rows *sql.Rows, maxRows int, scrollTo int) (return_rows []map[string]interface{}, column_types []ColumnType) {
	fields, _ := rows.Columns()

	colch := make(chan []ColumnType)
	defer close(colch)

	var wg sync.WaitGroup
	wg.Add(1)
	go prepareColumnTypes(*rows, &wg, colch)

	rowch := make(chan map[string]interface{})
	defer close(rowch)

	rowCount := 0

	//fmt.Println("ToMap2 rows.JumpToRow2(3)", rows.JumpToRow2(scrollTo))

	for rows.Next() {
		//rows.JumpToRow(3)

		rowCount += 1
		scans := make([]interface{}, len(fields))

		for i := range scans {
			scans[i] = &scans[i]
		}

		rows.Scan(scans...)

		go processRow2(scans, fields, rowch, &wg)
		if rowCount >= maxRows {
			break
		}
	}

	wg.Wait()

	column_types = <-colch

	for i := 1; i <= rowCount; i++ {
		row := <-rowch

		return_rows = append(return_rows, row)

	}

	return
}
