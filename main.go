package main

import (
	"log"
)

func main() {
	engine := NewEngine(NewRelationGraph(), map[string]*Policy{})
	service := NewService(engine)
	if err := service.Run(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
