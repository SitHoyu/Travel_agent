package local

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/travel-agent/shared/contracts"
)

func decodeGeneratePlanRequest(value interface{}) (contracts.GeneratePlanRequest, error) {
	switch v := value.(type) {
	case string:
		return decodeGeneratePlanRequestString(v)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return contracts.GeneratePlanRequest{}, fmt.Errorf("marshal request: %w", err)
		}
		return decodeGeneratePlanRequestBytes(raw)
	}
}

func decodeGeneratePlanRequestString(raw string) (contracts.GeneratePlanRequest, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return contracts.GeneratePlanRequest{}, fmt.Errorf("request string is empty")
	}
	return decodeGeneratePlanRequestBytes([]byte(trimmed))
}

func decodeGeneratePlanRequestBytes(raw []byte) (contracts.GeneratePlanRequest, error) {
	var req contracts.GeneratePlanRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return contracts.GeneratePlanRequest{}, fmt.Errorf("decode request: %w", err)
	}
	return req, nil
}
