package postgres

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
	templates.GenerateRSAKeyPair(dir)
	templates.GenerateCmd(module, dir)
	templates.GenerateConfig(dir)
	generateMigrations(dir)
	generateEnvs(dir)
	generateApi(module, dir, "message")
	generateStartup(module, dir, "message")
	generatePostgresInit(dir)
	generateDocker(dir)
	templates.ExecuteTidy(dir)
}

func generateMigrations(dir string) {
	d := filepath.Join(dir, "migrations")
	templates.CreateDir(d)
}

func generatePostgresInit(dir string) {
	base := filepath.Base(dir)
	d := filepath.Join(dir, ".extra", "setup")
	templates.CreateDir(d)

	pgseed := `-- write your seed sql queries here to populate initial schema in the database`

	initTestDb := fmt.Sprintf(`-- Create test user
CREATE USER %s_test_db_user WITH PASSWORD 'changeit';

-- Create test database
CREATE DATABASE %s_test_db OWNER %s_test_db_user;

GRANT ALL PRIVILEGES ON DATABASE %s_test_db TO %s_test_db_user;
`, base, base, base, base, base)

	templates.CreateFile(filepath.Join(d, "pgseed.sql"), pgseed)
	templates.CreateFile(filepath.Join(d, "init-test-db.sql"), initTestDb)
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
      - '${SERVER_PORT}:${SERVER_PORT}'
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    networks:
      - %s-network
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  postgres:
    image: postgres:18.1
    restart: unless-stopped
    env_file: .env
    ports:
      - '${DB_PORT}:5432'
    volumes:
      - dbdata:/data/db
      # optional pg seed scripts
      - ./.extra/setup/init-test-db.sql:/docker-entrypoint-initdb.d/init-test-db.sql:ro
      - ./.extra/setup/pgseed.sql:/docker-entrypoint-initdb.d/pgseed.sql:ro
    networks:
      - %s-network
    healthcheck:
      test:
        [
          "CMD-SHELL",
          "pg_isready -h localhost -p 5432 -U \"$${POSTGRES_USER}\" -d \"$${POSTGRES_DB}\""
        ]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 20s

  redis:
    image: redis:8.4.0
    restart: unless-stopped
    env_file: .env
    ports:
      - '${REDIS_PORT}:6379'
    command: redis-server --bind 0.0.0.0 --save 20 1 --loglevel warning --requirepass ${REDIS_PASSWORD}
    volumes:
      - cache:/data/cache
    networks:
      - %s-network
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

  migrate:
    image: migrate/migrate
    env_file: .test.env
    volumes:
      - ./migrations:/migrations
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - %s-network
    entrypoint: ["/bin/sh", "-c"]
    command:
      - |
        migrate -path /migrations -database "postgres://$${DB_USER}:$${DB_USER_PWD}@postgres:5432/$${DB_NAME}?sslmode=disable" up

networks:
  %s-network:
    driver: bridge

volumes:
  dbdata:
  cache:
    driver: local
`, base, base, base, base, base, base)

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

	mdl := fmt.Sprintf(`package startup

import (
	"context"

	coreMW "github.com/afteracademy/goserve/v2/middleware"
	"github.com/afteracademy/goserve/v2/postgres"
	"github.com/afteracademy/goserve/v2/network"
	"github.com/afteracademy/goserve/v2/redis"
	"%s/api/%s"
	"%s/config"
)

type Module network.Module[module]

type module struct {
	Context context.Context
	Env     *config.Env
	DB      postgres.Database
	Store   redis.Store
}

func (m *module) GetInstance() *module {
	return m
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

func NewModule(context context.Context, env *config.Env, db postgres.Database, store redis.Store) Module {
	return &module{
		Context: context,
		Env:     env,
		DB:      db,
		Store:   store,
	}
}
`, module, feature, module, feature, feature)

	server := fmt.Sprintf(`package startup

import (
	"context"
	"time"

	"github.com/afteracademy/goserve/v2/postgres"
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

	dbConfig := postgres.DbConfig{
		User:        env.DBUser,
		Pwd:         env.DBUserPwd,
		Host:        env.DBHost,
		Port:        env.DBPort,
		Name:        env.DBName,
		MinPoolSize: env.DBMinPoolSize,
		MaxPoolSize: env.DBMaxPoolSize,
		Timeout:     time.Duration(env.DBQueryTimeout) * time.Second,
	}

	db := postgres.NewDatabase(context, dbConfig)
	db.Connect()

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

	templates.CreateFile(filepath.Join(d, "module.go"), mdl)
	templates.CreateFile(filepath.Join(d, "server.go"), server)
	templates.CreateFile(filepath.Join(d, "testserver.go"), testServer)
}

func generateApi(module, dir, feature string) {
	d := filepath.Join(dir, "api")
	templates.CreateDir(d)
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
	"time"

	"github.com/google/uuid"
)

type Info%s struct {
	ID        uuid.UUID ` + "`" + `json:"_id" binding:"required"` + "`" + `
	Field     string    ` + "`" + `json:"field" binding:"required"` + "`" + `
	CreatedAt time.Time ` + "`" + `json:"createdAt" binding:"required"` + "`" + `
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
	featureCaps := capitalizeFirstLetter(featureName)
	modelPath := filepath.Join(featureDir, fmt.Sprintf("model/%s.go", featureLower))

	tStr := `package model

import (
	"time"

	"github.com/google/uuid"
)

type %s struct {
	ID        uuid.UUID  // id 
	Field     string     // field
	Status    bool       // status
	CreatedAt time.Time  // created_at
	UpdatedAt time.Time  // updated_at
}
`
	template := fmt.Sprintf(tStr, featureCaps)

	return os.WriteFile(modelPath, []byte(template), os.ModePerm)
}

func generateService(module, featureDir, featureName string) error {
	featureLower := strings.ToLower(featureName)
	featureCaps := capitalizeFirstLetter(featureName)
	servicePath := filepath.Join(featureDir, fmt.Sprintf("%sservice.go", ""))

	template := fmt.Sprintf(`package %s

import (
	"context"
  "%s/api/%s/dto"
	"%s/api/%s/model"

	"github.com/afteracademy/goserve/v2/redis"
	"github.com/afteracademy/goserve/v2/postgres"
	"github.com/google/uuid"
)

type Service interface {
	Find%s(id uuid.UUID) (*model.%s, error)
}

type service struct {
	db          postgres.Database
	info%sCache redis.Cache[dto.Info%s]
}

func NewService(db postgres.Database, store redis.Store) Service {
	return &service{
		db:          db,
		info%sCache: redis.NewCache[dto.Info%s](store),
	}
}

func (s *service) Find%s(id uuid.UUID) (*model.%s, error) {
	ctx := context.Background()

	query := `+"`"+`
		SELECT
			id,
			field,
			status,
			created_at,
			updated_at
		FROM %ss
		WHERE id = $1
	`+"`"+`

	var m model.%s

	err := s.db.Pool().QueryRow(ctx, query, id).
		Scan(
			&m.ID,
			&m.Field,
			&m.Status,
			&m.CreatedAt,
			&m.UpdatedAt,
		)

	if err != nil {
		return nil, err
	}

	return &m, nil
}
`, featureLower, module, featureLower, module, featureLower, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureCaps, featureLower, featureCaps)

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
	uuidParam, err := network.ReqParams[coredto.UUID](ctx)
	if err != nil {
		network.SendBadRequestError(ctx, err.Error(), err)
		return
	}

	%s, err := c.service.Find%s(uuidParam.ID)
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
DB_HOST=postgres
DB_PORT=5432
DB_NAME=%s_dev_db
DB_USER=%s_dev_db_user
DB_USER_PWD=changeit
DB_MIN_POOL_SIZE=2
DB_MAX_POOL_SIZE=5
DB_QUERY_TIMEOUT_SEC=60

# PostgreSQL Docker container variables
POSTGRES_DB=%s_dev_db
POSTGRES_USER=%s_dev_db_user
POSTGRES_PASSWORD=changeit

# REDIS_HOST=localhost
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=changeit

# 2 DAYS: 172800 Sec
ACCESS_TOKEN_VALIDITY_SEC=172800
# 7 DAYS: 604800 Sec
REFRESH_TOKEN_VALIDITY_SEC=604800
TOKEN_ISSUER=api.%s.com
TOKEN_AUDIENCE=%s.com

RSA_PRIVATE_KEY_PATH="keys/private.pem"
RSA_PUBLIC_KEY_PATH="keys/public.pem"
`, base, base, base, base, base, base)

	testEnv := fmt.Sprintf(`# debug, release, test
GO_MODE=debug

SERVER_HOST=0.0.0.0
SERVER_PORT=8081

# DB_HOST=localhost
DB_HOST=postgres
DB_PORT=5432
DB_NAME=%s_test_db
DB_USER=%s_test_db_user
DB_USER_PWD=changeit
DB_MIN_POOL_SIZE=2
DB_MAX_POOL_SIZE=5
DB_QUERY_TIMEOUT_SEC=60

# REDIS_HOST=localhost
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=changeit

# 2 DAYS: 172800 Sec
ACCESS_TOKEN_VALIDITY_SEC=172800
# 7 DAYS: 604800 Sec
REFRESH_TOKEN_VALIDITY_SEC=604800
TOKEN_ISSUER=api.%s.com
TOKEN_AUDIENCE=%s.com

# test run from the test directory one level below the src
RSA_PRIVATE_KEY_PATH="../keys/private.pem"
RSA_PUBLIC_KEY_PATH="../keys/public.pem"
`, base, base, base, base)

	templates.CreateFile(filepath.Join(dir, ".env"), env)
	templates.CreateFile(filepath.Join(dir, ".test.env"), testEnv)
}
