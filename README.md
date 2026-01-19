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

### B. Or You can use goservegen binary to generate the project
1. Download the goservegen binary for your operating system from the goservegen latest release: [github.com/afteracademy/goservegen/releases](https://github.com/afteracademy/goservegen/releases)

2. Expand the compressed file (Example: Apple Mac M2: goservegen_Darwin_arm64.tar.gz)

3. Run the binary 
```bash
cd ~/Downloads/goservegen_Darwin_arm64

# ./goservegen [project directory path] [project module] [Database Type - mongo/postgres]
./goservegen ~/Downloads/example github.com/yourusername/example postgres
```
> Note: `./goservegen ~/Downloads/example github.com/yourusername/example` will generate project named `example` located at `~/Downloads` and module `github.com/yourusername/example`

4. Open the generated project in your IDE/editor of choice

5. Have fun developing your REST API server!

## Generated Postgres Project
```
.
├── Dockerfile
├── api
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

## Generated Mongo Project
```
.
├── Dockerfile
├── api
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
├── startup
│   ├── indexes.go
│   ├── module.go
│   ├── server.go
│   └── testserver.go
└── utils
    ├── convertor.go
    └── file.go
```

## Run the project using Docker
```bash
docker compose up --build
```

Response
```
{
    "code": "10000",
    "status": 200,
    "message": "pong!"
}
```

## Working on the project
You can read about using this framework here [github.com/afteracademy/goserve](https://github.com/afteracademy/goserve)

## Read the Article to understand this project
[How to Architect Good Go Backend REST API Services](https://afteracademy.com/article/how-to-architect-good-go-backend-rest-api-services)

## Troubleshoot
Sometimes your operating system will block the binary from execution, you will have to provide permission to run it. 

> In Mac you have to go System Settings > Privacy & Security > Allow goservegen