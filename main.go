package main

import (
	"log"
	"os"

	"github.com/afteracademy/goservegen/templates/mongo"
	"github.com/afteracademy/goservegen/templates/postgres"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("project name is required")
	}

	if len(os.Args[1]) == 0 {
		log.Fatalln("project name should be non-empty string")
	}

	if len(os.Args) < 3 {
		log.Fatalln("project module name is required")
	}

	if len(os.Args[2]) == 0 {
		log.Fatalln("project module name should be non-empty string")
	}

	if len(os.Args) < 4 {
		log.Fatalln("project type 'mongo or postgres' is required")
	}

	if len(os.Args[3]) == 0 {
		log.Fatalln("project type 'mongo or postgres' should be non-empty string")
	}

	if os.Args[3] != "mongo" && os.Args[3] != "postgres" {
		log.Fatalln("project type must be 'mongo or postgres' only")
	}

	switch os.Args[3] {
	case "mongo":
		mongo.Generate(os.Args[1], os.Args[2])
	case "postgres":
		postgres.Generate(os.Args[1], os.Args[2])
	default:
		log.Fatalln("unsupported project type")
	}
}
