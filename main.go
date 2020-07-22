package main

import (
	"flag"
	"fmt"
)

var managerFlag = flag.Bool("a", false, "run as admin-side")

func main() {

	fmt.Println("Copyright 2019-2020 hyperchain.net (Hyperchain)")
	fmt.Println("HC admin tool")

	//flag.PrintDefaults()
	flag.Parse()
	if *managerFlag {
		fmt.Println("Run as admin-side")
		adminmain()
	} else {
		fmt.Println("Run as monitored-side")
		clientmain()
	}
}
