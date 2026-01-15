package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
		log.Fatalln("project name should be non-empty string")
	}

	dir := os.Args[1]
	module := os.Args[2]

	createDir(dir)
	generateGoMod(module, dir)
	generateEnvs(dir)
	generateIgnores(dir)
	generateUtils(dir)
	generateConfig(dir)
	generateRSAKeyPair(dir)
	generateApi(module, dir, "sample")
	generateStartup(module, dir, "sample")
	generateCmd(module, dir)
	generateMongoInit(dir)
	generateDocker(dir)
	executeTidy(dir)
}

func createDir(dir string) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Fatalf("error creating directory: %s", dir)
	}
}

func createFile(file, content string) {
	if err := os.WriteFile(file, []byte(content), os.ModePerm); err != nil {
		log.Fatalf("error creating file: %s", file)
	}
}

func executeTidy(dir string) {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Command execution failed: %v\nOutput: %s", err, string(output))
	}
}

func generateMongoInit(dir string) {
	d := filepath.Join(dir, ".extra", "setup")
	createDir(d)

	initMongo := `function seed(dbName, user, password) {
  db = db.getSiblingDB(dbName);
  db.createUser({
    user: user,
    pwd: password,
    roles: [{ role: "readWrite", db: dbName }],
  });
}

seed("dev-db", "dev-db-user", "changeit");
seed("test-db", "test-db-user", "changeit");
`
	createFile(filepath.Join(d, "init-mongo.js"), initMongo)
}

func generateDocker(dir string) {
	base := filepath.Base(dir)
	docker := fmt.Sprintf(`FROM golang:1.25.5-alpine

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

	compose := `services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    image: api
    container_name: api
    restart: unless-stopped
    env_file: .env
    ports:
      - '${SERVER_PORT}:8080'
    depends_on:
      - mongo
      - redis

  mongo:
    image: mongo:8.0.9
    container_name: mongo
    restart: unless-stopped
    env_file: .env
    environment:
      - MONGO_INITDB_ROOT_USERNAME=${DB_ADMIN}
      - MONGO_INITDB_ROOT_PASSWORD=${DB_ADMIN_PWD}
      - MONGO_INITDB_DATABASE=${DB_NAME}
    ports:
      - '${DB_PORT}:27017'
    command: mongod --bind_ip_all
    volumes:
      - ./.extra/setup/init-mongo.js:/docker-entrypoint-initdb.d/init-mongo.js:ro
      - dbdata:/data/db

  redis:
    image: redis:8.0.0
    container_name: redis
    restart: unless-stopped
    env_file: .env
    ports:
      - '${REDIS_PORT}:6379'
    command: redis-server --bind localhost --bind 0.0.0.0 --save 20 1 --loglevel warning --requirepass ${REDIS_PASSWORD}
    volumes:
      - cache:/data/cache

volumes:
  dbdata:
  cache:
    driver: local
`

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
	createFile(filepath.Join(dir, "Dockerfile"), docker)
	createFile(filepath.Join(dir, "docker-compose.yml"), compose)
	createFile(filepath.Join(dir, ".dockerignore"), ignore)
}

func generateCmd(module, dir string) {
	d := filepath.Join(dir, "cmd")
	createDir(d)

	m := fmt.Sprintf(`package main

import "%s/startup"

func main() {
	startup.Server()
}
`, module)

	createFile(filepath.Join(d, "main.go"), m)
}

func generateStartup(module, dir, feature string) {
	d := filepath.Join(dir, "startup")
	createDir(d)

	indexes := fmt.Sprintf(`package startup

import (
	"github.com/afteracademy/goserve/arch/mongo"
	%sModel "%s/api/sample/model"
)

func EnsureDbIndexes(db mongo.Database) {
	go mongo.Document[sampleModel.Sample](&sampleModel.Sample{}).EnsureIndexes(db)
}
`, feature, module)

	mdl := fmt.Sprintf(`package startup

import (
	"context"

	coreMW "github.com/afteracademy/goserve/arch/middleware"
	"github.com/afteracademy/goserve/arch/mongo"
	"github.com/afteracademy/goserve/arch/network"
	"github.com/afteracademy/goserve/arch/redis"
	"%s/api/sample"
	"%s/config"
)

type Module network.Module[module]

type module struct {
	Context context.Context
	Env     *config.Env
	DB      mongo.Database
	Store   redis.Store
}

func (m *module) GetInstance() *module {
	return m
}

func (m *module) Controllers() []network.Controller {
	return []network.Controller{
		sample.NewController(m.AuthenticationProvider(), m.AuthorizationProvider(), sample.NewService(m.DB, m.Store)),
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
	return &module{
		Context: context,
		Env:     env,
		DB:      db,
		Store:   store,
	}
}
`, module, module)

	server := fmt.Sprintf(`package startup

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/afteracademy/goserve/arch/mongo"
	"github.com/afteracademy/goserve/arch/network"
	"github.com/afteracademy/goserve/arch/redis"
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

	"github.com/afteracademy/goserve/arch/network"
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

	createFile(filepath.Join(d, "indexes.go"), indexes)
	createFile(filepath.Join(d, "module.go"), mdl)
	createFile(filepath.Join(d, "server.go"), server)
	createFile(filepath.Join(d, "testserver.go"), testServer)
}

func generateApi(module, dir, feature string) {
	d := filepath.Join(dir, "api")
	createDir(d)
	generateApiFeature(module, d, feature)
}

func capitalizeFirstLetter(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(string(str[0])) + str[1:]
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
	featureCaps := capitalizeFirstLetter(featureName)
	dtoPath := filepath.Join(featureDir, fmt.Sprintf("dto/create_%s.go", featureLower))

	tStr := `package dto

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Info%s struct {
	ID        primitive.ObjectID ` + "`" + `json:"_id" binding:"required"` + "`" + `
	Field     string             ` + "`" + `json:"field" binding:"required"` + "`" + `
	CreatedAt time.Time          ` + "`" + `json:"createdAt" binding:"required"` + "`" + `
}

func EmptyInfo%s() *Info%s {
	return &Info%s{}
}

func (d *Info%s) GetValue() *Info%s {
	return d
}

func (d *Info%s) ValidateErrors(errs validator.ValidationErrors) ([]string, error) {
	var msgs []string
	for _, err := range errs {
		switch err.Tag() {
		case "required":
			msgs = append(msgs, fmt.Sprintf("%%s is required", err.Field()))
		case "min":
			msgs = append(msgs, fmt.Sprintf("%%s must be min %%s", err.Field(), err.Param()))
		case "max":
			msgs = append(msgs, fmt.Sprintf("%%s must be max %%s", err.Field(), err.Param()))
		default:
			msgs = append(msgs, fmt.Sprintf("%%s is invalid", err.Field()))
		}
	}
	return msgs, nil
}
`
	template := fmt.Sprintf(tStr, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps)

	return os.WriteFile(dtoPath, []byte(template), os.ModePerm)
}

func generateModel(featureDir, featureName string) error {
	modelDirPath := filepath.Join(featureDir, "model")
	if err := os.MkdirAll(modelDirPath, os.ModePerm); err != nil {
		return err
	}

	featureLower := strings.ToLower(featureName)
	featureCaps := capitalizeFirstLetter(featureName)
	modelPath := filepath.Join(featureDir, fmt.Sprintf("model/%s.go", featureLower))

	tStr := `package model

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/afteracademy/goserve/arch/mongo"
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

func (doc *%s) GetValue() *%s {
	return doc
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
	template := fmt.Sprintf(tStr, featureLower, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps)

	return os.WriteFile(modelPath, []byte(template), os.ModePerm)
}

func generateService(module, featureDir, featureName string) error {
	featureLower := strings.ToLower(featureName)
	featureCaps := capitalizeFirstLetter(featureName)
	servicePath := filepath.Join(featureDir, fmt.Sprintf("%sservice.go", ""))

	template := fmt.Sprintf(`package %s

import (
  "%s/api/%s/dto"
	"%s/api/%s/model"
	"github.com/afteracademy/goserve/arch/mongo"
	"github.com/afteracademy/goserve/arch/network"
	"github.com/afteracademy/goserve/arch/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Service interface {
	Find%s(id primitive.ObjectID) (*model.%s, error)
}

type service struct {
	network.BaseService
	%sQueryBuilder mongo.QueryBuilder[model.%s]
	info%sCache    redis.Cache[dto.Info%s]
}

func NewService(db mongo.Database, store redis.Store) Service {
	return &service{
		BaseService:  network.NewBaseService(),
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
`, featureLower, module, featureLower, module, featureLower, featureCaps, featureCaps, featureLower, featureCaps, featureCaps, featureCaps, featureLower, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureLower)

	return os.WriteFile(servicePath, []byte(template), os.ModePerm)
}

func generateController(module, featureDir, featureName string) error {
	featureLower := strings.ToLower(featureName)
	featureCaps := capitalizeFirstLetter(featureName)
	controllerPath := filepath.Join(featureDir, fmt.Sprintf("%scontroller.go", ""))

	template := fmt.Sprintf(`package %s

import (
	"github.com/gin-gonic/gin"
	"%s/api/%s/dto"
	coredto "github.com/afteracademy/goserve/arch/dto"
	"github.com/afteracademy/goserve/arch/network"
	"%s/utils"
)

type controller struct {
	network.BaseController
	service Service
}

func NewController(
	authMFunc network.AuthenticationProvider,
	authorizeMFunc network.AuthorizationProvider,
	service Service,
) network.Controller {
	return &controller{
		BaseController: network.NewBaseController("/%s", authMFunc, authorizeMFunc),
		service:  service,
	}
}

func (c *controller) MountRoutes(group *gin.RouterGroup) {
group.GET("/ping", c.getPingHandler)
	group.GET("/id/:id", c.get%sHandler)
}

func (c *controller) getPingHandler(ctx *gin.Context) {
	c.Send(ctx).SuccessMsgResponse("pong!")
}

func (c *controller) get%sHandler(ctx *gin.Context) {
	mongoId, err := network.ReqParams(ctx, coredto.EmptyMongoId())
	if err != nil {
		c.Send(ctx).BadRequestError(err.Error(), err)
		return
	}

	%s, err := c.service.Find%s(mongoId.ID)
	if err != nil {
		c.Send(ctx).NotFoundError("%s not found", err)
		return
	}

	data, err := utils.MapTo[dto.Info%s](%s)
	if err != nil {
		c.Send(ctx).InternalServerError("something went wrong", err)
		return
	}

	c.Send(ctx).SuccessDataResponse("success", data)
}
`, featureLower, module, featureLower, module, featureLower, featureCaps, featureCaps, featureLower, featureCaps, featureLower, featureCaps, featureLower)

	return os.WriteFile(controllerPath, []byte(template), os.ModePerm)
}

func generateRSAKeyPair(dir string) error {
	d := filepath.Join(dir, "keys")
	createDir(d)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	if err := privateKey.Validate(); err != nil {
		return err
	}

	privatePemBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	private, err := os.Create(filepath.Join(dir, "keys", "private.pem"))
	if err != nil {
		return err
	}
	defer private.Close()

	if err := pem.Encode(private, privatePemBlock); err != nil {
		return err
	}

	derBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	publicPemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: derBytes,
	}

	public, err := os.Create(filepath.Join(dir, "keys", "public.pem"))
	if err != nil {
		return err
	}
	defer public.Close()

	if err := pem.Encode(public, publicPemBlock); err != nil {
		return err
	}

	return nil
}

func generateConfig(dir string) {
	env := `package config

import (
	"log"

	"github.com/spf13/viper"
)

type Env struct {
	// server
	GoMode     string ` + "`" + `mapstructure:"GO_MODE"` + "`" + `
	ServerHost string ` + "`" + `mapstructure:"SERVER_HOST"` + "`" + `
	ServerPort uint16 ` + "`" + `mapstructure:"SERVER_PORT"` + "`" + `
	// database
	DBHost         string ` + "`" + `mapstructure:"DB_HOST"` + "`" + `
	DBName         string ` + "`" + `mapstructure:"DB_NAME"` + "`" + `
	DBPort         uint16 ` + "`" + `mapstructure:"DB_PORT"` + "`" + `
	DBUser         string ` + "`" + `mapstructure:"DB_USER"` + "`" + `
	DBUserPwd      string ` + "`" + `mapstructure:"DB_USER_PWD"` + "`" + `
	DBMinPoolSize  uint16 ` + "`" + `mapstructure:"DB_MIN_POOL_SIZE"` + "`" + `
	DBMaxPoolSize  uint16 ` + "`" + `mapstructure:"DB_MAX_POOL_SIZE"` + "`" + `
	DBQueryTimeout uint16 ` + "`" + `mapstructure:"DB_QUERY_TIMEOUT_SEC"` + "`" + `
	// redis
	RedisHost string ` + "`" + `mapstructure:"REDIS_HOST"` + "`" + `
	RedisPort uint16 ` + "`" + `mapstructure:"REDIS_PORT"` + "`" + `
	RedisPwd  string ` + "`" + `mapstructure:"REDIS_PASSWORD"` + "`" + `
	RedisDB   int    ` + "`" + `mapstructure:"REDIS_DB"` + "`" + `
	// keys
	RSAPrivateKeyPath string ` + "`" + `mapstructure:"RSA_PRIVATE_KEY_PATH"` + "`" + `
	RSAPublicKeyPath  string ` + "`" + `mapstructure:"RSA_PUBLIC_KEY_PATH"` + "`" + `
	// Token
	AccessTokenValiditySec  uint64 ` + "`" + `mapstructure:"ACCESS_TOKEN_VALIDITY_SEC"` + "`" + `
	RefreshTokenValiditySec uint64 ` + "`" + `mapstructure:"REFRESH_TOKEN_VALIDITY_SEC"` + "`" + `
	TokenIssuer             string ` + "`" + `mapstructure:"TOKEN_ISSUER"` + "`" + `
	TokenAudience           string ` + "`" + `mapstructure:"TOKEN_AUDIENCE"` + "`" + `
}

func NewEnv(filename string, override bool) *Env {
	env := Env{}
	viper.SetConfigFile(filename)

	if override {
		viper.AutomaticEnv()
	}

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal("Error reading environment file", err)
	}

	err = viper.Unmarshal(&env)
	if err != nil {
		log.Fatal("Error loading environment file", err)
	}

	return &env
}
`
	d := filepath.Join(dir, "config")
	createDir(d)
	createFile(filepath.Join(d, "env.go"), env)
}

func generateUtils(dir string) {
	convertor := `package utils

import (
	"github.com/jinzhu/copier"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func IsValidObjectID(id string) bool {
	_, err := primitive.ObjectIDFromHex(id)
	return err == nil
}

func MapTo[T any, V any](from *V) (*T, error) {
	var to T
	err := copier.Copy(&to, from)
	if err != nil {
		return nil, err
	}
	return &to, nil
}
`
	d := filepath.Join(dir, "utils")
	createDir(d)
	createFile(filepath.Join(d, "convertor.go"), convertor)
}

func generateIgnores(dir string) {
	gitignore := `
 # If you prefer the allow list template instead of the deny list, see community template:
# https://github.com/github/gitignore/blob/main/community/Golang/Go.AllowList.gitignore
#
.DS_Store
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with ` + "`" + `go test -c` + "`" + `
*.test
!Dockerfile.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories (remove the comment below to include it)
# vendor/

# Go workspace file
go.work
go.work.sum

# Environment varibles
*.env
*.env.test

#keys
keys/*
!keys/*.md
!keys/*.txt
*.pem

__debug*

build
 `
	createFile(filepath.Join(dir, ".gitignore"), gitignore)
}

func generateEnvs(dir string) {
	env := `
	# debug, release, test
GO_MODE=debug

SERVER_HOST=0.0.0.0
SERVER_PORT=8080

DB_HOST=mongo
# DB_HOST=localhost
DB_PORT=27017
DB_NAME=dev-db
DB_USER=dev-db-user
DB_USER_PWD=changeit
DB_MIN_POOL_SIZE=2
DB_MAX_POOL_SIZE=5
DB_QUERY_TIMEOUT_SEC=60
DB_ADMIN=admin
DB_ADMIN_PWD=changeit

REDIS_HOST=redis
# REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=changeit

# 2 DAYS: 172800 Sec
ACCESS_TOKEN_VALIDITY_SEC=172800
# 7 DAYS: 604800 Sec
REFRESH_TOKEN_VALIDITY_SEC=604800
TOKEN_ISSUER=api.goserve.afteracademy.com
TOKEN_AUDIENCE=goserve.afteracademy.com

RSA_PRIVATE_KEY_PATH="keys/private.pem"
RSA_PUBLIC_KEY_PATH="keys/public.pem"
`

	testEnv := `
# debug, release, test
GO_MODE=test

SERVER_HOST=0.0.0.0
SERVER_PORT=8080

DB_HOST=mongo
# DB_HOST=localhost
DB_PORT=27017
DB_NAME=test-db
DB_USER=test-db-user
DB_USER_PWD=changeit
DB_MIN_POOL_SIZE=2
DB_MAX_POOL_SIZE=5
DB_QUERY_TIMEOUT_SEC=60
DB_ADMIN=admin
DB_ADMIN_PWD=changeit

REDIS_HOST=redis
# REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=changeit

# 2 DAYS: 172800 Sec
ACCESS_TOKEN_VALIDITY_SEC=172800
# 7 DAYS: 604800 Sec
REFRESH_TOKEN_VALIDITY_SEC=604800
TOKEN_ISSUER=api.goserve.afteracademy.com
TOKEN_AUDIENCE=goserve.afteracademy.com

RSA_PRIVATE_KEY_PATH="../keys/private.pem"
RSA_PUBLIC_KEY_PATH="../keys/public.pem"
`

	createFile(filepath.Join(dir, ".env"), env)
	createFile(filepath.Join(dir, ".test.env"), testEnv)
}

func generateGoMod(module, dir string) {
	goMod := `module %s

go 1.25.5

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/go-playground/validator/v10 v10.22.0
	github.com/jinzhu/copier v0.4.0
	github.com/spf13/viper v1.19.0
	github.com/afteracademy/goserve v1.1.9
	go.mongodb.org/mongo-driver v1.15.1
)

require (
	github.com/bytedance/sonic v1.11.9 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.4 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/redis/go-redis/v9 v9.5.3 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240424034433-3c2c7870ae76 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
`

	goSum := `module %s

go 1.25.5

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/go-playground/validator/v10 v10.22.0
	github.com/jinzhu/copier v0.4.0
	github.com/spf13/viper v1.19.0
	github.com/afteracademy/goserve v1.1.9
	go.mongodb.org/mongo-driver v1.15.1
)

require (
	github.com/bytedance/sonic v1.11.9 // indirect
	github.com/bytedance/sonic/loader v0.1.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.4 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.4 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/redis/go-redis/v9 v9.5.3 // indirect
	github.com/sagikazarmark/locafero v0.6.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.12 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240424034433-3c2c7870ae76 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/arch v0.8.0 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
`
	createFile(filepath.Join(dir, "go.mod"), fmt.Sprintf(goMod, module))
	createFile(filepath.Join(dir, "go.sum"), fmt.Sprintf(goSum, module))
}
