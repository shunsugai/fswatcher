package main

import (
	"fmt"
	"github.com/fatih/color"
	"os"
)

func cprintln(a ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s ", color.YellowString(APP_NAME), color.GreenString(">>>"))
	fmt.Println(a...)
}

func cfatal(a ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s ERROR: ", color.YellowString(APP_NAME), color.GreenString(">>>"))
	fmt.Println(a...)
	os.Exit(1)
}
