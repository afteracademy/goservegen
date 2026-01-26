# goservegen - Go Backend Architecture Generator using goserve framework
[![Download](https://img.shields.io/badge/Download-Starter%20Project%20Mongo%20Zip-green.svg)](https://github.com/afteracademy/goservegen/raw/main/starter-project-mongo.zip)
[![Download](https://img.shields.io/badge/Download-Starter%20Project%20Postgres%20Zip-green.svg)](https://github.com/afteracademy/goservegen/raw/main/starter-project-postgres.zip)

Project generator for go backend architecture using goserve framework

## See more on goserve framework
[github.com/afteracademy/goserve](https://github.com/afteracademy/goserve)

## Check the example project built using goservegen
1. [goserve-example-api-server-mongo](https://github.com/afteracademy/goserve-example-api-server-mongo)

2. [goserve-example-api-server-postgres](https://github.com/afteracademy/goserve-example-api-server-postgres)

## How To Use goservegen

### A. Either You can download the starter projects here
1. [Starter Project Mongo Zip](https://github.com/afteracademy/goservegen/raw/main/starter-project-mongo.zip)
2. [Starter Project Postgres Zip](https://github.com/afteracademy/goservegen/raw/main/starter-project-postgres.zip)	

### B. Or You can use goservegen directly to generate the project
Install go language in your system if not already installed. [Download Go](https://go.dev/dl/)

### goservegen [project directory path] [project module] [Database Type - mongo/postgres]
Postgres Project
```bash
go run github.com/afteracademy/goservegen/v2@latest ~/Downloads/my_project github.com/yourusername/example postgres
```

Mongo Project
```bash
go run github.com/afteracademy/goservegen/v2@latest ~/Downloads/my_project github.com/yourusername/example mongo
```

> Note: It will generate project named `my_project` located at `~/Downloads` and module `github.com/yourusername/example`

## Run the project using Docker
```bash
# Go to the project directory
cd ~/Downloads/my_project	
```

```bash
docker compose up --build
```

## Healthy Check
```bash
# Run on terminal
curl http://localhost:8080/health
```

Response
```json
{
  "code": "10000",
  "status": 200,
  "message": "success",
  "data": {
    "timestamp": "2026-01-25T06:45:17.228713387Z",
    "status": "OK"
  }
}
```

### Now Open the generated project in your IDE/editor of choice
> Have fun developing your REST API server!

## Generated Project Structure
```
.
├── Dockerfile
├── api
│   ├── health
│   │   ├── controller.go
│   │   ├── dto
│   │   │   └── health_check.go
│   │   └── service.go
│   └── message
│       ├── controller.go
│       ├── dto
│       │   └── create_message.go
│       ├── model
│       │   └── message.go
│       └── service.go
├── cmd
│   └── main.go
├── config
│   └── env.go
├── docker-compose.yml
├── go.mod
├── go.sum
├── keys
│   ├── private.pem
│   └── public.pem
├── migrations
├── startup
│   ├── module.go
│   ├── server.go
│   └── testserver.go
└── utils
    ├── convertor.go
    └── file.go
```

## Working on the project
You can read about using this framework here [github.com/afteracademy/goserve](https://github.com/afteracademy/goserve)

## Read the Article to understand this project
[How to Architect Good Go Backend REST API Services](https://afteracademy.com/article/how-to-architect-good-go-backend-rest-api-services)
