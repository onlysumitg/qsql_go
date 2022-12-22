package main

// -----------------------------------------------------------------
//
// -----------------------------------------------------------------
func (app *application) batches() {
	if app.batchSQLModel != nil {
		go app.batchSQLModel.BatchProcess().Run()
	}
}
