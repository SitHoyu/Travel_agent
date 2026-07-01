package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/travel-agent/services/plan-orchestrator/internal/agent"
	authutil "github.com/travel-agent/services/plan-orchestrator/internal/auth"
	"github.com/travel-agent/services/plan-orchestrator/internal/client/amapgeo"
	"github.com/travel-agent/services/plan-orchestrator/internal/client/amaphotel"
	"github.com/travel-agent/services/plan-orchestrator/internal/client/amapweather"
	"github.com/travel-agent/services/plan-orchestrator/internal/client/llmgateway"
	appconfig "github.com/travel-agent/services/plan-orchestrator/internal/config"
	"github.com/travel-agent/services/plan-orchestrator/internal/controller"
	httpHandler "github.com/travel-agent/services/plan-orchestrator/internal/handler/http"
	"github.com/travel-agent/services/plan-orchestrator/internal/middleware"
	"github.com/travel-agent/services/plan-orchestrator/internal/orchestrator"
	planrepo "github.com/travel-agent/services/plan-orchestrator/internal/repository/plan"
	userrepo "github.com/travel-agent/services/plan-orchestrator/internal/repository/user"
	authservice "github.com/travel-agent/services/plan-orchestrator/internal/service/auth"
	mysqlstorage "github.com/travel-agent/services/plan-orchestrator/internal/storage/mysql"
	"github.com/travel-agent/services/plan-orchestrator/internal/toolkit"
	"github.com/travel-agent/services/plan-orchestrator/internal/toolkit/local"
)

func main() {
	cfg, err := appconfig.Load("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	llmClient := llmgateway.NewClient(cfg.LLMGateway.BaseURL, cfg.LLMGateway.Provider, cfg.LLMGateway.Model)
	geoClient := amapgeo.NewClient(cfg.AMap.BaseURL, cfg.AMap.APIKey)
	hotelClient := amaphotel.NewClient(cfg.AMap.BaseURL, cfg.AMap.APIKey)
	weatherClient := amapweather.NewClient(cfg.AMap.BaseURL, cfg.AMap.APIKey)
	cityResolver, err := local.LoadCityCodeResolver(cfg.AMap.AdcodeFile)
	if err != nil {
		log.Fatalf("load amap adcode file: %v", err)
	}
	locationEnricher := local.NewLocationEnricher(geoClient)

	toolRegistry := toolkit.NewRegistry()
	toolRegistry.Register(local.NewThinkTool())
	toolRegistry.Register(local.NewWeatherTool(weatherClient, cityResolver))
	toolRegistry.Register(local.NewBuildItineraryDraftTool(llmClient, locationEnricher))
	toolRegistry.Register(local.NewValidateConstraintsTool())
	toolRegistry.Register(local.NewRecommendHotelAreaTool(hotelClient))

	runtimeAgent := agent.NewLLMAgent(llmClient, toolRegistry)
	controller := controller.New(runtimeAgent, toolRegistry, cfg.Controller.MaxSteps)
	planRepository, userRepository, err := buildRepositories(cfg)
	if err != nil {
		log.Fatalf("init repositories: %v", err)
	}
	planService := orchestrator.NewService(controller, planRepository)
	tokenManager := authutil.NewTokenManager(cfg.Auth.JWTSecret, time.Duration(cfg.Auth.TokenTTLHours)*time.Hour)
	userService := authservice.NewService(userRepository, tokenManager)
	handler := httpHandler.NewHandler(planService, userService, middleware.AuthRequired(tokenManager))

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), middleware.CORS())
	handler.RegisterRoutes(router)

	addr := ":" + cfg.Server.Port
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("plan-orchestrator listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}

func buildRepositories(cfg appconfig.Config) (planrepo.Repository, userrepo.Repository, error) {
	switch cfg.Storage.Driver {
	case "", "memory":
		return planrepo.NewInMemoryRepository(), userrepo.NewInMemoryRepository(), nil
	case "mysql":
		db, err := mysqlstorage.Open(cfg.Storage.DSN)
		if err != nil {
			return nil, nil, err
		}
		return planrepo.NewMySQLRepository(db), userrepo.NewMySQLRepository(db), nil
	default:
		return nil, nil, logError("unsupported storage driver %q", cfg.Storage.Driver)
	}
}

func logError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
