package templates

import (
	"fmt"
	"os"
	"path/filepath"
)

func GenerateHealthApi(module, dir string) error {
	featureDir := filepath.Join(dir, "health")

	if err := os.MkdirAll(featureDir, os.ModePerm); err != nil {
		return err
	}

	if err := generateDto(featureDir); err != nil {
		return err
	}
	if err := generateService(module, featureDir); err != nil {
		return err
	}
	if err := generateController(featureDir); err != nil {
		return err
	}
	return nil
}
func generateDto(featureDir string) error {
	dtoDirPath := filepath.Join(featureDir, "dto")
	if err := os.MkdirAll(dtoDirPath, os.ModePerm); err != nil {
		return err
	}

	dtoPath := filepath.Join(featureDir, "dto/health_check.go")

	tStr := `package dto

import (
	"time"
)

type HealthCheck struct {
	Timestamp time.Time` + "`" + `json:"timestamp" binding:"required"` + "`" + `
	Status    string    ` + "`" + `json:"status" binding:"required"` + "`" + `
}
`
	return os.WriteFile(dtoPath, []byte(tStr), os.ModePerm)
}

func generateService(module, featureDir string) error {
	servicePath := filepath.Join(featureDir, "service.go")

	template := fmt.Sprintf(`package health

import (
	"time"

	"%s/api/health/dto"
)

type Service interface {
	CheckHealth() (*dto.HealthCheck, error)
}

type service struct {
}

func NewService() Service {
	return &service{}
}

func (s *service) CheckHealth() (*dto.HealthCheck, error) {
	health := &dto.HealthCheck{
		Timestamp: time.Now(),
		Status:    "OK",
	}
	return health, nil
}
`, module)

	return os.WriteFile(servicePath, []byte(template), os.ModePerm)
}

func generateController(featureDir string) error {
	controllerPath := filepath.Join(featureDir, "controller.go")

	template := `package health

import (
	"github.com/afteracademy/goserve/v2/network"
	"github.com/gin-gonic/gin"
)

type controller struct {
	network.Controller
	service Service
}

func NewController(
	service Service,
) network.Controller {
	return &controller{
		Controller: network.NewController("/health", nil, nil),
		service:    service,
	}
}

func (c *controller) MountRoutes(group *gin.RouterGroup) {
	group.GET("", c.getHealthHandler)
}

func (c *controller) getHealthHandler(ctx *gin.Context) {
	health, err := c.service.CheckHealth()
	if err != nil {
		network.SendMixedError(ctx, err)
		return
	}

	network.SendSuccessDataResponse(ctx, "success", health)
}

`

	return os.WriteFile(controllerPath, []byte(template), os.ModePerm)
}
