package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	appconfig "github.com/travel-agent/services/llm-gateway/internal/config"
	httpHandler "github.com/travel-agent/services/llm-gateway/internal/handler/http"
	"github.com/travel-agent/services/llm-gateway/internal/llm"
	"github.com/travel-agent/services/llm-gateway/internal/service"
)

func main() {
	cfg, err := appconfig.Load("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	promptStore, err := llm.NewPromptStore(cfg.Prompts.BaseDir)
	if err != nil {
		log.Fatalf("init prompt store: %v", err)
	}

	registry, err := llm.NewRegistry(cfg.Providers)
	if err != nil {
		log.Fatalf("init provider registry: %v", err)
	}

	svc := service.New(promptStore, registry)
	handler := httpHandler.NewHandler(svc)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	handler.RegisterRoutes(router)

	addr := ":" + cfg.Server.Port
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	log.Printf("llm-gateway listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
