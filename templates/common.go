package templates

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
)

const GO_VERSION = "1.25.5"

func CreateDir(dir string) {
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		log.Fatalf("error creating directory: %s", dir)
	}
}

func CreateFile(file, content string) {
	if err := os.WriteFile(file, []byte(content), os.ModePerm); err != nil {
		log.Fatalf("error creating file: %s", file)
	}
}

func GenerateGoMod(module, dir string) {
	goMod := `module %s

go ` + GO_VERSION + `

`

	CreateFile(filepath.Join(dir, "go.mod"), fmt.Sprintf(goMod, module))
}

func GenerateIgnores(dir string) {
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
	CreateFile(filepath.Join(dir, ".gitignore"), gitignore)
}

func GenerateUtils(dir string) {
	convertor := `package utils

import (
	"strconv"
	"strings"
)

func ConvertUint16(str string) uint16 {
	u, err := strconv.ParseUint(str, 10, 16)
	if err != nil {
		return 0
	}
	return uint16(u)
}

func ConvertUint8(str string) uint8 {
	u, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		return 0
	}
	return uint8(u)
}

func ExtractBearerToken(authHeader string) string {
	const prefix = "Bearer "
	tokenIndex := strings.Index(authHeader, prefix)
	if tokenIndex == -1 || tokenIndex != 0 {
		return ""
	}
	return authHeader[tokenIndex+len(prefix):]
}

func FormatEndpoint(endpoint string) string {
	endpoint = strings.ReplaceAll(endpoint, " ", "")
	endpoint = strings.ReplaceAll(endpoint, "/", "-")
	endpoint = strings.ReplaceAll(endpoint, "?", "")
	return endpoint
}
`

	file := `package utils

import "os"

func LoadPEMFileInto(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}
`

	d := filepath.Join(dir, "utils")
	CreateDir(d)
	CreateFile(filepath.Join(d, "convertor.go"), convertor)
	CreateFile(filepath.Join(d, "file.go"), file)
}

func GenerateRSAKeyPair(dir string) error {
	d := filepath.Join(dir, "keys")
	CreateDir(d)

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

func GenerateCmd(module, dir string) {
	d := filepath.Join(dir, "cmd")
	CreateDir(d)

	m := fmt.Sprintf(`package main

import "%s/startup"

func main() {
	startup.Server()
}
`, module)

	CreateFile(filepath.Join(d, "main.go"), m)
}

func GenerateConfig(dir string) {
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
	CreateDir(d)
	CreateFile(filepath.Join(d, "env.go"), env)
}

func ExecuteTidy(dir string) {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Command execution failed: %v\nOutput: %s", err, string(output))
	}
}
