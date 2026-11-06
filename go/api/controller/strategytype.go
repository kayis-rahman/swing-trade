package controller

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/kayis-rahman/swing-trade/go/core/clock"
	strategy "github.com/kayis-rahman/swing-trade/go/strategy"
)

// strategyTypeResponse mirrors the Java StrategyTypeController.StrategyTypeResponse.
type strategyTypeResponse struct {
	RequiredIndicators []string            `json:"requiredIndicators"`
	ParamSchema        paramSchemaResponse `json:"paramSchema"`
	Type               string              `json:"type"`
	WarmupBars         *int                `json:"warmupBars"`
}

type paramSchemaResponse struct {
	Params []paramDefResponse `json:"params"`
}

type paramDefResponse struct {
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	Min          *string `json:"min"`
	Max          *string `json:"max"`
	DefaultValue any     `json:"defaultValue"`
	Description  string  `json:"description"`
	Group        string  `json:"group"`
}

// RegisterStrategyTypeRoutes mirrors Java's StrategyTypeController.
func RegisterStrategyTypeRoutes(mux *http.ServeMux, registry *strategy.StrategyTypeRegistry, _ strategy.ParamSchemaValidator, _ *slog.Logger, clk clock.Clock) {
	if registry == nil {
		registry = strategy.NewStrategyTypeRegistry(
			strategy.NewLegacyPriceActionAdapter(strategy.PriceActionStrategy{}),
			strategy.NewPullbackStrategy(),
			strategy.NewSqueezeStrategy(),
			strategy.IntradayOpeningRangeBreakoutStrategy{},
			strategy.IntradayVwapPullbackStrategy{},
			strategy.IntradayRsi2Strategy{},
		)
	}
	mux.HandleFunc("GET /api/strategy-types", func(w http.ResponseWriter, r *http.Request) {
		strategies := registry.All()
		sort.Slice(strategies, func(i, j int) bool { return strategies[i].Type() < strategies[j].Type() })
		result := make([]strategyTypeResponse, 0, len(strategies))
		for _, s := range strategies {
			validator := strategy.ParamSchemaValidator{}
			defaultResult := validator.Validate(s.ParamSchema(), s.CrossFieldRules(), map[string]any{})
			resp := strategyTypeResponse{
				Type:               s.Type(),
				ParamSchema:        buildParamSchema(s.ParamSchema()),
				RequiredIndicators: []string{},
			}
			if defaultResult.Valid {
				params := strategy.StrategyParamsViewOf(defaultResult.ResolvedParams)
				warmup := s.WarmupBars(params)
				resp.WarmupBars = &warmup
				indicators := s.RequiredIndicators(params)
				names := make([]string, 0, len(indicators))
				for _, indicator := range indicators {
					names = append(names, indicator.String())
				}
				sort.Strings(names)
				resp.RequiredIndicators = names
			} else {
				empty := []string{}
				resp.RequiredIndicators = empty
			}
			result = append(result, resp)
		}
		writeResponseJSON(w, http.StatusOK, apiEnvelope{Data: result, Error: nil, Success: true, Timestamp: javaLocalDateTime(clk.Now())})
	})
}

func buildParamSchema(schema strategy.ParamSchema) paramSchemaResponse {
	response := paramSchemaResponse{Params: make([]paramDefResponse, 0, len(schema.Params))}
	for _, def := range schema.Params {
		item := paramDefResponse{
			Name:         def.Name,
			Type:         string(def.Type),
			Description:  def.Description,
			Group:        def.Group,
			DefaultValue: def.DefaultValue,
		}
		if def.Min != nil {
			value := def.Min.String()
			item.Min = &value
		}
		if def.Max != nil {
			value := def.Max.String()
			item.Max = &value
		}
		response.Params = append(response.Params, item)
	}
	return response
}
