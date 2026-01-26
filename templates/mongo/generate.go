package mongo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/afteracademy/goservegen/v2/templates"
)

func Generate(dir string, module string) {
	templates.CreateDir(dir)
	templates.GenerateGoMod(module, dir)
	templates.GenerateIgnores(dir)
	templates.GenerateUtils(dir)
	templates.GenerateConfig(dir)
	templates.GenerateRSAKeyPair(dir)
	templates.GenerateCmd(module, dir)
	generateApi(module, dir, "message")
	generateStartup(module, dir, "message")
	generateEnvs(dir)
	generateMongoInit(dir)
	generateDocker(dir)
	templates.ExecuteTidy(dir)
}

func generateMongoInit(dir string) {
	base := filepath.Base(dir)
	d := filepath.Join(dir, ".extra", "setup")
	templates.CreateDir(d)

	initMongo := fmt.Sprintf(`function seed(dbName, user, password) {
  db = db.getSiblingDB(dbName);
  db.createUser({
    user: user,
    pwd: password,
    roles: [{ role: "readWrite", db: dbName }],
  });
}

seed("%s-dev-db", "%s-dev-db-user", "changeit");
seed("%s-test-db", "%s-test-db-user", "changeit");
`, base, base, base, base)

	templates.CreateFile(filepath.Join(d, "init-mongo.js"), initMongo)
}

func generateDocker(dir string) {
	base := filepath.Base(dir)
	docker := fmt.Sprintf(`FROM golang:`+templates.GO_VERSION+`-alpine

RUN apk add --no-cache curl

RUN adduser --disabled-password --gecos '' gouser

RUN mkdir -p /home/gouser/%s

WORKDIR /home/gouser/%s

COPY . .

RUN chown -R gouser:gouser /home/gouser/%s

USER gouser

RUN go mod tidy
RUN go build -o build/server cmd/main.go

EXPOSE 8080

CMD ["./build/server"]
 `, base, base, base)

	compose := fmt.Sprintf(`services:
  %s:
    build:
      context: .
      dockerfile: Dockerfile
    restart: unless-stopped
    env_file: .env
    ports:
      - '${SERVER_PORT}:8080'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    depends_on:
      mongo:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - %s-network

  mongo:
    image: mongo:8.0.9
    restart: unless-stopped
    env_file: .env
    ports:
      - '${DB_PORT}:27017'
    command: mongod --bind_ip_all
    volumes:
      - ./.extra/setup/init-mongo.js:/docker-entrypoint-initdb.d/init-mongo.js:ro
      - dbdata:/data/db
    healthcheck:
      test:
        [
          "CMD",
          "mongosh",
          "--quiet",
          "-u", "${MONGO_INITDB_ROOT_USERNAME}",
          "-p", "${MONGO_INITDB_ROOT_PASSWORD}",
          "--authenticationDatabase", "admin",
          "--eval", "db.runCommand({ ping: 1 }).ok"
        ]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 20s
    networks:
      - %s-network

  redis:
    image: redis:8.4.0
    restart: unless-stopped
    env_file: .env
    ports:
      - '${REDIS_PORT}:6379'
    command: redis-server --bind localhost --bind 0.0.0.0 --save 20 1 --loglevel warning --requirepass ${REDIS_PASSWORD}
    volumes:
      - cache:/data/cache
    healthcheck:
      test:
        [
          "CMD",
          "redis-cli",
          "-a", "${REDIS_PASSWORD}",
          "ping"
        ]
      interval: 10s
      timeout: 3s
      retries: 5
      start_period: 10s
    networks:
      - %s-network

networks:
  %s-network:
    driver: bridge

volumes:
  dbdata:
  cache:
    driver: local
`, base, base, base, base, base)

	ignore := `
# Binaries
/server
/server.exe

# Vendor directory (if not using Go modules)
vendor/

# OS-specific files
*.exe
*.dll
*.so
*.dylib

# Test output
*.out

# Logs
*.log

# Coverage files
*.cover
*.coverage
*.cov

# Build directories
bin/
obj/
build/
dist/

# IDE/editor directories and files
.vscode/
.idea/
*.swp
*~

# Git
.git/
.gitignore

# Docker
.dockerignore
Dockerfile

# Dependency management files
go.sum

# Any other files you want to exclude
.DS_Store 
.github/
.tools/
logs/
*.md
`
	templates.CreateFile(filepath.Join(dir, "Dockerfile"), docker)
	templates.CreateFile(filepath.Join(dir, "docker-compose.yml"), compose)
	templates.CreateFile(filepath.Join(dir, ".dockerignore"), ignore)
}

func generateStartup(module, dir, feature string) {
	d := filepath.Join(dir, "startup")
	templates.CreateDir(d)
	featureCaps := templates.CapitalizeFirstLetter(feature)

	indexes := fmt.Sprintf(`package startup

import (
	"github.com/afteracademy/goserve/v2/mongo"
	%sModel "%s/api/%s/model"
)

func EnsureDbIndexes(db mongo.Database) {
	go mongo.Document[%sModel.%s](&%sModel.%s{}).EnsureIndexes(db)
}
`, feature, module, feature, feature, featureCaps, feature, featureCaps)

	mdl := fmt.Sprintf(`package startup

import (
	"context"

	coreMW "github.com/afteracademy/goserve/v2/middleware"
	"github.com/afteracademy/goserve/v2/mongo"
	"github.com/afteracademy/goserve/v2/network"
	"github.com/afteracademy/goserve/v2/redis"
	"%s/api/health"
	"%s/api/%s"
	"%s/config"
)

type Module network.Module[module]

type module struct {
	Context context.Context
	Env     *config.Env
	DB      mongo.Database
	Store   redis.Store
	HealthService health.Service
}

func (m *module) GetInstance() *module {
	return m
}

// OpenControllers are controllers that do not require api key authentication
func (m *module) OpenControllers() []network.Controller {
	return []network.Controller{health.NewController(m.HealthService)}
}

func (m *module) Controllers() []network.Controller {
	return []network.Controller{
		%s.NewController(m.AuthenticationProvider(), m.AuthorizationProvider(), %s.NewService(m.DB, m.Store)),
	}
}

func (m *module) RootMiddlewares() []network.RootMiddleware {
	return []network.RootMiddleware{
		coreMW.NewErrorCatcher(),
		coreMW.NewNotFound(),
	}
}

func (m *module) AuthenticationProvider() network.AuthenticationProvider {
	// TODO
	return nil
}

func (m *module) AuthorizationProvider() network.AuthorizationProvider {
	// TODO
	return nil
}

func NewModule(context context.Context, env *config.Env, db mongo.Database, store redis.Store) Module {
	healthService := health.NewService()
	return &module{
		Context: context,
		Env:     env,
		DB:      db,
		Store:   store,
		HealthService: healthService,
	}
}
`, module, module, feature, module, feature, feature)

	server := fmt.Sprintf(`package startup

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/afteracademy/goserve/v2/mongo"
	"github.com/afteracademy/goserve/v2/network"
	"github.com/afteracademy/goserve/v2/redis"
	"%s/config"
)

type Shutdown = func()

func Server() {
	env := config.NewEnv(".env", true)
	router, _, shutdown := create(env)
	defer shutdown()
	router.Start(env.ServerHost, env.ServerPort)
}

func create(env *config.Env) (network.Router, Module, Shutdown) {
	context := context.Background()

	dbConfig := mongo.DbConfig{
		User:        env.DBUser,
		Pwd:         env.DBUserPwd,
		Host:        env.DBHost,
		Port:        env.DBPort,
		Name:        env.DBName,
		MinPoolSize: env.DBMinPoolSize,
		MaxPoolSize: env.DBMaxPoolSize,
		Timeout:     time.Duration(env.DBQueryTimeout) * time.Second,
	}

	db := mongo.NewDatabase(context, dbConfig)
	db.Connect()

	if env.GoMode != gin.TestMode {
		EnsureDbIndexes(db)
	}

	redisConfig := redis.Config{
		Host: env.RedisHost,
		Port: env.RedisPort,
		Pwd:  env.RedisPwd,
		DB:   env.RedisDB,
	}

	store := redis.NewStore(context, &redisConfig)
	store.Connect()

	module := NewModule(context, env, db, store)

	router := network.NewRouter(env.GoMode)
	router.RegisterValidationParsers(network.CustomTagNameFunc())
	router.LoadControllers(module.GetInstance().OpenControllers())
	router.LoadRootMiddlewares(module.RootMiddlewares())
	router.LoadControllers(module.Controllers())

	shutdown := func() {
		db.Disconnect()
		store.Disconnect()
	}

	return router, module, shutdown
}
`, module)

	testServer := fmt.Sprintf(`package startup

import (
	"net/http/httptest"

	"github.com/afteracademy/goserve/v2/network"
	"%s/config"
)

type Teardown = func()

func TestServer() (network.Router, Module, Teardown) {
	env := config.NewEnv("../.test.env", false)
	router, module, shutdown := create(env)
	ts := httptest.NewServer(router.GetEngine())
	teardown := func() {
		ts.Close()
		shutdown()
	}
	return router, module, teardown
}
`, module)

	templates.CreateFile(filepath.Join(d, "indexes.go"), indexes)
	templates.CreateFile(filepath.Join(d, "module.go"), mdl)
	templates.CreateFile(filepath.Join(d, "server.go"), server)
	templates.CreateFile(filepath.Join(d, "testserver.go"), testServer)
}

func generateApi(module, dir, feature string) {
	d := filepath.Join(dir, "api")
	templates.CreateDir(d)
	generateApiFeature(module, d, feature)
	templates.GenerateHealthApi(module, d)
}

func generateApiFeature(module, dir, feature string) error {
	featureName := strings.ToLower(feature)
	featureDir := filepath.Join(dir, featureName)

	if err := os.MkdirAll(featureDir, os.ModePerm); err != nil {
		return err
	}

	if err := generateDto(featureDir, featureName); err != nil {
		return err
	}
	if err := generateModel(featureDir, featureName); err != nil {
		return err
	}
	if err := generateService(module, featureDir, featureName); err != nil {
		return err
	}
	if err := generateController(module, featureDir, featureName); err != nil {
		return err
	}
	return nil
}

func generateDto(featureDir, featureName string) error {
	dtoDirPath := filepath.Join(featureDir, "dto")
	if err := os.MkdirAll(dtoDirPath, os.ModePerm); err != nil {
		return err
	}

	featureLower := strings.ToLower(featureName)
	featureCaps := templates.CapitalizeFirstLetter(featureName)
	dtoPath := filepath.Join(featureDir, fmt.Sprintf("dto/create_%s.go", featureLower))

	tStr := `package dto

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Info%s struct {
	ID        primitive.ObjectID ` + "`" + `json:"_id" binding:"required"` + "`" + `
	Field     string             ` + "`" + `json:"field" binding:"required"` + "`" + `
	CreatedAt time.Time          ` + "`" + `json:"createdAt" binding:"required"` + "`" + `
}
`
	template := fmt.Sprintf(tStr, featureCaps)

	return os.WriteFile(dtoPath, []byte(template), os.ModePerm)
}

func generateModel(featureDir, featureName string) error {
	modelDirPath := filepath.Join(featureDir, "model")
	if err := os.MkdirAll(modelDirPath, os.ModePerm); err != nil {
		return err
	}

	featureLower := strings.ToLower(featureName)
	featureCaps := templates.CapitalizeFirstLetter(featureName)
	modelPath := filepath.Join(featureDir, fmt.Sprintf("model/%s.go", featureLower))

	tStr := `package model

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/afteracademy/goserve/v2/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	mongod "go.mongodb.org/mongo-driver/mongo"
)

const CollectionName = "%ss"

type %s struct {
	ID        primitive.ObjectID ` + "`" + `bson:"_id,omitempty" validate:"-"` + "`" + `
	Field     string             ` + "`" + `bson:"field" validate:"required"` + "`" + `
	Status    bool               ` + "`" + `bson:"status" validate:"required"` + "`" + `
	CreatedAt time.Time          ` + "`" + `bson:"createdAt" validate:"required"` + "`" + `
	UpdatedAt time.Time          ` + "`" + `bson:"updatedAt" validate:"required"` + "`" + `
}` + `

func New%s(field string) (*%s, error) {
	time := time.Now()
	doc := %s{
		Field:     field,
		Status:    true,
		CreatedAt: time,
		UpdatedAt: time,
	}
	if err := doc.Validate(); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (doc *%s) Validate() error {
	validate := validator.New()
	return validate.Struct(doc)
}

func (*%s) EnsureIndexes(db mongo.Database) {
	indexes := []mongod.IndexModel{
		{
			Keys: bson.D{
				{Key: "_id", Value: 1},
				{Key: "status", Value: 1},
			},
		},
	}
	
	mongo.NewQueryBuilder[%s](db, CollectionName).Query(context.Background()).CreateIndexes(indexes)
}

`
	template := fmt.Sprintf(tStr, featureLower, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps)

	return os.WriteFile(modelPath, []byte(template), os.ModePerm)
}

func generateService(module, featureDir, featureName string) error {
	featureLower := strings.ToLower(featureName)
	featureCaps := templates.CapitalizeFirstLetter(featureName)
	servicePath := filepath.Join(featureDir, fmt.Sprintf("%sservice.go", ""))

	template := fmt.Sprintf(`package %s

import (
  "%s/api/%s/dto"
	"%s/api/%s/model"
	"github.com/afteracademy/goserve/v2/mongo"
	"github.com/afteracademy/goserve/v2/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	Find%s(id primitive.ObjectID) (*model.%s, error)
}

type service struct {
	%sQueryBuilder mongo.QueryBuilder[model.%s]
	info%sCache    redis.Cache[dto.Info%s]
}

func NewService(db mongo.Database, store redis.Store) Service {
	return &service{
		%sQueryBuilder: mongo.NewQueryBuilder[model.%s](db, model.CollectionName),
		info%sCache: redis.NewCache[dto.Info%s](store),
	}
}

func (s *service) Find%s(id primitive.ObjectID) (*model.%s, error) {
	filter := bson.M{"_id": id}

	msg, err := s.%sQueryBuilder.SingleQuery().FindOne(filter, nil)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
`, featureName, module, featureLower, module, featureLower, featureCaps, featureCaps, featureLower, featureCaps, featureCaps, featureCaps, featureLower, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureLower)

	return os.WriteFile(servicePath, []byte(template), os.ModePerm)
}

func generateController(module, featureDir, featureName string) error {
	featureLower := strings.ToLower(featureName)
	featureCaps := templates.CapitalizeFirstLetter(featureName)
	controllerPath := filepath.Join(featureDir, fmt.Sprintf("%scontroller.go", ""))

	template := fmt.Sprintf(`package %s

import (
	"github.com/gin-gonic/gin"
	"%s/api/%s/dto"
	coredto "github.com/afteracademy/goserve/v2/dto"
	"github.com/afteracademy/goserve/v2/network"
	"github.com/afteracademy/goserve/v2/utility"
)

type controller struct {
	network.Controller
	service Service
}

func NewController(
	authMFunc network.AuthenticationProvider,
	authorizeMFunc network.AuthorizationProvider,
	service Service,
) network.Controller {
	return &controller{
		Controller: network.NewController("/%s", authMFunc, authorizeMFunc),
		service:  service,
	}
}

func (c *controller) MountRoutes(group *gin.RouterGroup) {
	group.GET("/id/:id", c.get%sHandler)
}

func (c *controller) get%sHandler(ctx *gin.Context) {
	mongoId, err := network.ReqParams[coredto.MongoId](ctx)
	if err != nil {
		network.SendBadRequestError(ctx, err.Error(), err)
		return
	}

	%s, err := c.service.Find%s(mongoId.ID)
	if err != nil {
		network.SendBadRequestError(ctx, err.Error(), err)
		return
	}

	data, err := utility.MapTo[dto.Info%s](%s)
	if data == nil || err != nil {
		network.SendBadRequestError(ctx, err.Error(), err)
		return
	}

	network.SendSuccessDataResponse(ctx, "success", data)
}
`, featureName, module, featureLower, featureLower, featureCaps, featureCaps, featureLower, featureCaps, featureCaps, featureLower)

	return os.WriteFile(controllerPath, []byte(template), os.ModePerm)
}

func generateEnvs(dir string) {
	base := filepath.Base(dir)
	env := fmt.Sprintf(`# debug, release, test
GO_MODE=debug

SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# DB_HOST=localhost
DB_HOST=mongo
DB_PORT=27017
DB_NAME=%s-dev-db
DB_USER=%s-dev-db-user
DB_USER_PWD=changeit
DB_MIN_POOL_SIZE=2
DB_MAX_POOL_SIZE=5
DB_QUERY_TIMEOUT_SEC=60

# MongoDB Admin Credentials
MONGO_INITDB_ROOT_USERNAME=admin
MONGO_INITDB_ROOT_PASSWORD=changeit
MONGO_INITDB_DATABASE=admin

# REDIS_HOST=localhost
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=changeit

# 2 DAYS: 172800 Sec
ACCESS_TOKEN_VALIDITY_SEC=172800
# 7 DAYS: 604800 Sec
REFRESH_TOKEN_VALIDITY_SEC=604800
TOKEN_ISSUER=api.%s.com
TOKEN_AUDIENCE=goserve.%s.com

RSA_PRIVATE_KEY_PATH="keys/private.pem"
RSA_PUBLIC_KEY_PATH="keys/public.pem"
`, base, base, base, base)

	testEnv := fmt.Sprintf(`# debug, release, test
GO_MODE=test

# DB_HOST=localhost
DB_HOST=mongo
DB_PORT=27017
DB_NAME=%s-test-db
DB_USER=%s-test-db-user
DB_USER_PWD=changeit
DB_MIN_POOL_SIZE=2
DB_MAX_POOL_SIZE=5
DB_QUERY_TIMEOUT_SEC=60
DB_ADMIN=admin
DB_ADMIN_PWD=changeit

# REDIS_HOST=localhost
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=changeit

# 2 DAYS: 172800 Sec
ACCESS_TOKEN_VALIDITY_SEC=172800
# 7 DAYS: 604800 Sec
REFRESH_TOKEN_VALIDITY_SEC=604800
TOKEN_ISSUER=api.%s.com
TOKEN_AUDIENCE=goserve.%s.com

RSA_PRIVATE_KEY_PATH="../keys/private.pem"
RSA_PUBLIC_KEY_PATH="../keys/public.pem"
`, base, base, base, base)

	templates.CreateFile(filepath.Join(dir, ".env"), env)
	templates.CreateFile(filepath.Join(dir, ".test.env"), testEnv)
}
