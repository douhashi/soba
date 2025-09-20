package main

import (
	"flag"
	"fmt"

	"github.com/douhashi/soba/internal/greeting"
)

func main() {
	var (
		name     string
		japanese bool
	)

	flag.StringVar(&name, "name", "", "Name to greet")
	flag.BoolVar(&japanese, "ja", false, "Use Japanese greeting")
	flag.Parse()

	if japanese {
		fmt.Println(greeting.JapaneseGreeting(name))
	} else {
		fmt.Println(greeting.Hello(name))
	}
}