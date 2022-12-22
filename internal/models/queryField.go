package models

// -----------------------------------------------------------------
// -----------------------------------------------------------------
// Define a new User type. Notice how the field names and types align
// with the columns in the database "users" table?
type QueryField struct {
	ID           string
	Name         string
	DefaultValue string
}
