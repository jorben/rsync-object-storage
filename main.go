package main

import (
	"fmt"
	"github.com/jorben/rsync-object-storage/config"
	"log"
)

func main() {

	c, err := config.GetConfig()
	if err != nil {
		log.Fatalf("load config err: %s\n", err.Error())
	}

	fmt.Printf("%v\n", c)

}
