package main

import (
	"flag"
	"fmt"
	"time"
)

func main() {
	flag.Parse()
	t, err := time.Parse("2006-01", flag.Args()[0])
	if err != nil {
		panic(err)
	}
	monthEnd := t.AddDate(0, 1, -1) // add a month, minus a day
	for i := 0; i < monthEnd.Day(); i++ {
		fmt.Println(t.AddDate(0, 0, i).Format("2006-01-02 Mon"))
	}
}
