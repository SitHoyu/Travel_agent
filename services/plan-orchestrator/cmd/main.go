package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/travel-agent/services/plan-orchestrator/internal/agent"
	"github.com/travel-agent/services/plan-orchestrator/internal/client/amapweather"
	"github.com/travel-agent/services/plan-orchestrator/internal/client/llmgateway"
	appconfig "github.com/travel-agent/services/plan-orchestrator/internal/config"
	"github.com/travel-agent/services/plan-orchestrator/internal/controller"
	httpHandler "github.com/travel-agent/services/plan-orchestrator/internal/handler/http"
	"github.com/travel-agent/services/plan-orchestrator/internal/orchestrator"
	planrepo "github.com/travel-agent/services/plan-orchestrator/internal/repository/plan"
	"github.com/travel-agent/services/plan-orchestrator/internal/toolkit"
	"github.com/travel-agent/services/plan-orchestrator/internal/toolkit/local"
)

func main() {
	cfg, err := appconfig.Load("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	llmClient := llmgateway.NewClient(cfg.LLMGateway.BaseURL, cfg.LLMGateway.Provider, cfg.LLMGateway.Model)
	weatherClient := amapweather.NewClient(cfg.AMap.BaseURL, cfg.AMap.APIKey)
	cityResolver, err := local.LoadCityCodeResolver(cfg.AMap.AdcodeFile)
	if err != nil {
		log.Fatalf("load amap adcode file: %v", err)
	}

	toolRegistry := toolkit.NewRegistry()
	toolRegistry.Register(local.NewThinkTool())
	toolRegistry.Register(local.NewWeatherTool(weatherClient, cityResolver))
	toolRegistry.Register(local.NewBuildItineraryDraftTool(llmClient))
	toolRegistry.Register(local.NewValidateConstraintsTool())

	runtimeAgent := agent.NewLLMAgent(llmClient, toolRegistry)
	controller := controller.New(runtimeAgent, toolRegistry, cfg.Controller.MaxSteps)
	repository := planrepo.NewInMemoryRepository()
	service := orchestrator.NewService(controller, repository)
	handler := httpHandler.NewHandler(service)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
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
