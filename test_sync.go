package main

import (
	"fmt"
	"os"
	"github.com/qdd-framework/qdd/pkg/qcl/graph"
)

func main() {
	cwd, _ := os.Getwd()
	gdb, err := graph.InitDB()
	if err != nil {
		panic(err)
	}
	err = gdb.SyncToGraph(cwd)
	if err != nil {
		panic(err)
	}
	fmt.Println("Synced successfully")
}
