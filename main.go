/*
Copyright 2026 Markus Papenbrock
*/
package main

import (
	"github.com/srlmgr/backend/cmd"
	_ "github.com/srlmgr/backend/services/importsvc/importer/acc"     // register acc processor
	_ "github.com/srlmgr/backend/services/importsvc/importer/iracing" // register iracing processor
)

func main() {
	cmd.Execute()
}
