package dashboard

import (
	"fmt"
	"log/slog"
	"math"

	"api-server/internal/app/dto"
)

// HoltWintersForecaster implements time series forecasting
type HoltWintersForecaster struct {
	logger *slog.Logger
}

// NewHoltWintersForecaster creates a new forecaster
func NewHoltWintersForecaster(logger *slog.Logger) *HoltWintersForecaster {
	return &HoltWintersForecaster{
		logger: logger,
	}
}

// ForecastConfig holds configuration for forecasting
type ForecastConfig struct {
	Alpha  float64 // Level smoothing factor (0-1)
	Beta   float64 // Trend smoothing factor (0-1)
	Gamma  float64 // Seasonal smoothing factor (0-1)
	Phi    float64 // Trend dampening factor (0.8-0.98, closer to 1 = less dampening)
	Period int     // Seasonal period
}

// DefaultForecastConfig returns scientifically validated defaults
// Based on M3-Competition findings (Makridakis & Hibon, 2000)
func DefaultForecastConfig() ForecastConfig {
	return ForecastConfig{
		Alpha:  0.3,  // Moderate level smoothing
		Beta:   0.1,  // Conservative trend smoothing
		Gamma:  0.05, // Minimal seasonality
		Phi:    0.90, // Standard dampening (research shows 0.85-0.98 works well)
		Period: 2,
	}
}

// ForecastResult holds the forecast results with confidence intervals
type ForecastResult struct {
	Values        []float64
	ConfidenceMin []float64
	ConfidenceMax []float64
	Method        string
}

// DampedHoltState holds the state for Damped Holt's method
type DampedHoltState struct {
	Level float64
	Trend float64
}

// initializeDampedHolt initializes level and trend using first few observations
// Using the recommended approach from Hyndman & Athanasopoulos (2021)
func (f *HoltWintersForecaster) initializeDampedHolt(data []float64) DampedHoltState {
	if len(data) < 2 {
		return DampedHoltState{Level: data[0], Trend: 0}
	}

	// Use regression on first few points for initialization
	n := min(len(data), 5)

	// Simple linear regression to get initial level and trend
	var sumX, sumY, sumXY, sumX2 float64
	for i := 0; i < n; i++ {
		x := float64(i)
		y := data[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	nFloat := float64(n)
	denominator := nFloat*sumX2 - sumX*sumX

	var trend float64
	if denominator != 0 {
		trend = (nFloat*sumXY - sumX*sumY) / denominator
	}

	level := sumY/nFloat - trend*sumX/nFloat

	// Ensure level is not negative
	if level < 0 {
		level = data[0]
	}

	return DampedHoltState{
		Level: level,
		Trend: trend,
	}
}

// DampedHoltForecast implements Damped Holt's Linear Trend method
// This is the scientifically recommended approach for business forecasting
// Reference: Gardner & McKenzie (1985), "Forecasting Trends in Time Series"
func (f *HoltWintersForecaster) DampedHoltForecast(data []float64, steps int, config ForecastConfig) ([]float64, DampedHoltState, error) {
	if len(data) < 2 {
		return nil, DampedHoltState{}, fmt.Errorf("need at least 2 data points")
	}

	// Initialize state
	state := f.initializeDampedHolt(data)

	// Fit the model to historical data
	for t := 1; t < len(data); t++ {
		y := data[t]

		// Update equations for Damped Holt's method:
		// L_t = α * Y_t + (1 - α) * (L_{t-1} + φ * b_{t-1})
		// b_t = β * (L_t - L_{t-1}) + (1 - β) * φ * b_{t-1}
		prevLevel := state.Level
		state.Level = config.Alpha*y + (1-config.Alpha)*(state.Level+config.Phi*state.Trend)
		state.Trend = config.Beta*(state.Level-prevLevel) + (1-config.Beta)*config.Phi*state.Trend
	}

	// Generate forecasts
	// F_{t+h} = L_t + (φ + φ² + ... + φ^h) * b_t
	// The sum φ + φ² + ... + φ^h = φ * (1 - φ^h) / (1 - φ)
	predictions := make([]float64, steps)
	for h := 1; h <= steps; h++ {
		var trendSum float64
		if config.Phi == 1.0 {
			trendSum = float64(h)
		} else {
			trendSum = config.Phi * (1 - math.Pow(config.Phi, float64(h))) / (1 - config.Phi)
		}
		predictions[h-1] = state.Level + trendSum*state.Trend

		// Ensure non-negative
		if predictions[h-1] < 0 {
			predictions[h-1] = 0
		}
	}

	return predictions, state, nil
}

// optimizeParameters finds optimal parameters using walk-forward validation
// This is a simplified grid search - production should use more sophisticated methods
func (f *HoltWintersForecaster) optimizeParameters(data []float64) ForecastConfig {
	if len(data) < 6 {
		return DefaultForecastConfig()
	}

	bestConfig := DefaultForecastConfig()
	bestMAPE := math.MaxFloat64

	// Grid search over parameter space
	// Research-backed ranges from forecasting literature
	alphas := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	betas := []float64{0.05, 0.1, 0.15, 0.2}
	phis := []float64{0.85, 0.90, 0.95, 0.98}

	// Use walk-forward validation: train on first 70%, validate on rest
	trainSize := int(float64(len(data)) * 0.7)
	if trainSize < 3 {
		trainSize = 3
	}
	testSize := len(data) - trainSize

	for _, alpha := range alphas {
		for _, beta := range betas {
			for _, phi := range phis {
				config := ForecastConfig{
					Alpha: alpha,
					Beta:  beta,
					Phi:   phi,
				}

				// Train and forecast
				trainData := data[:trainSize]
				predictions, _, err := f.DampedHoltForecast(trainData, testSize, config)
				if err != nil {
					continue
				}

				// Calculate MAPE on test set
				mape := f.calculateMAPE(data[trainSize:], predictions)
				if mape < bestMAPE {
					bestMAPE = mape
					bestConfig = config
				}
			}
		}
	}

	f.logger.Info("Parameter optimization completed",
		"alpha", bestConfig.Alpha,
		"beta", bestConfig.Beta,
		"phi", bestConfig.Phi,
		"validation_mape", fmt.Sprintf("%.2f%%", bestMAPE))

	return bestConfig
}

// calculateMAPE calculates Mean Absolute Percentage Error
func (f *HoltWintersForecaster) calculateMAPE(actual, predicted []float64) float64 {
	n := min(len(actual), len(predicted))
	if n == 0 {
		return math.MaxFloat64
	}

	var sumAPE float64
	count := 0
	for i := 0; i < n; i++ {
		if actual[i] != 0 {
			sumAPE += math.Abs((actual[i] - predicted[i]) / actual[i])
			count++
		}
	}

	if count == 0 {
		return math.MaxFloat64
	}
	return sumAPE / float64(count) * 100
}

// calculateForecastError estimates forecast error for confidence intervals
// Using the approach from Hyndman et al. (2008)
func (f *HoltWintersForecaster) calculateForecastError(data []float64, config ForecastConfig) float64 {
	if len(data) < 4 {
		// Default to 20% of mean if insufficient data
		var sum float64
		for _, v := range data {
			sum += v
		}
		return sum / float64(len(data)) * 0.20
	}

	// Calculate one-step-ahead forecast errors
	state := f.initializeDampedHolt(data)
	var sumSquaredError float64

	for t := 1; t < len(data); t++ {
		// One-step forecast
		forecast := state.Level + config.Phi*state.Trend
		error := data[t] - forecast
		sumSquaredError += error * error

		// Update state
		prevLevel := state.Level
		state.Level = config.Alpha*data[t] + (1-config.Alpha)*(state.Level+config.Phi*state.Trend)
		state.Trend = config.Beta*(state.Level-prevLevel) + (1-config.Beta)*config.Phi*state.Trend
	}

	// Return RMSE (Root Mean Square Error)
	return math.Sqrt(sumSquaredError / float64(len(data)-1))
}

// Forecast generates forecasts using the Damped Holt method with parameter optimization
func (f *HoltWintersForecaster) Forecast(data []float64, steps int, config ForecastConfig) (*ForecastResult, error) {
	if len(data) < 3 {
		return nil, fmt.Errorf("insufficient data points (minimum 3 required, got %d)", len(data))
	}

	f.logger.Info("Starting Damped Holt forecast",
		"data_points", len(data),
		"steps", steps)

	// Filter out leading zeros for cleaner estimation
	startIdx := 0
	for i, v := range data {
		if v > 0 {
			startIdx = i
			break
		}
	}
	cleanData := data[startIdx:]

	if len(cleanData) < 3 {
		cleanData = data
	}

	// Optimize parameters if we have enough data
	var optConfig ForecastConfig
	if len(cleanData) >= 6 {
		optConfig = f.optimizeParameters(cleanData)
	} else {
		optConfig = DefaultForecastConfig()
	}

	// Override with any non-default values from input config
	if config.Phi > 0 && config.Phi != DefaultForecastConfig().Phi {
		optConfig.Phi = config.Phi
	}

	// Generate forecast
	predictions, finalState, err := f.DampedHoltForecast(cleanData, steps, optConfig)
	if err != nil {
		return nil, fmt.Errorf("damped holt forecast failed: %w", err)
	}

	// Calculate forecast error for confidence intervals
	sigma := f.calculateForecastError(cleanData, optConfig)

	// Generate confidence intervals
	// Variance grows with forecast horizon for damped trend method
	confidenceMin := make([]float64, steps)
	confidenceMax := make([]float64, steps)

	for h := 0; h < steps; h++ {
		// Approximate prediction interval width
		// For damped trend: σ_h ≈ σ * sqrt(1 + h * (some factor based on α, β, φ))
		// Simplified approximation:
		horizonFactor := math.Sqrt(1 + float64(h)*0.1)
		interval := 1.96 * sigma * horizonFactor // 95% confidence

		confidenceMin[h] = math.Max(0, predictions[h]-interval)
		confidenceMax[h] = predictions[h] + interval
	}

	f.logger.Info("Damped Holt forecast completed",
		"final_level", finalState.Level,
		"final_trend", finalState.Trend,
		"first_forecast", predictions[0],
		"last_forecast", predictions[steps-1],
		"optimized_alpha", optConfig.Alpha,
		"optimized_beta", optConfig.Beta,
		"optimized_phi", optConfig.Phi)

	return &ForecastResult{
		Values:        predictions,
		ConfidenceMin: confidenceMin,
		ConfidenceMax: confidenceMax,
		Method:        "damped-holt",
	}, nil
}

// ForecastMultipleMetrics generates forecasts for multiple business metrics
func (f *HoltWintersForecaster) ForecastMultipleMetrics(historicalData dto.HistoricalData) (map[string]*ForecastResult, error) {
	results := make(map[string]*ForecastResult)
	config := DefaultForecastConfig()

	// Forecast weekly pay first (needed for capital calculation)
	var weeklyPayResult *ForecastResult
	if len(historicalData.WeeklyPayData) >= 3 {
		weeklyPayValues := make([]float64, len(historicalData.WeeklyPayData))
		for i, data := range historicalData.WeeklyPayData {
			weeklyPayValues[i] = data.TotalWeeklyPay
		}

		result, err := f.Forecast(weeklyPayValues, 24, config)
		if err != nil {
			f.logger.Error("Failed to forecast weekly pay", "error", err)
		} else {
			results["weekly_pay"] = result
			weeklyPayResult = result
		}

		// Forecast paid employees
		paidEmployeesValues := make([]float64, len(historicalData.WeeklyPayData))
		for i, data := range historicalData.WeeklyPayData {
			paidEmployeesValues[i] = float64(data.PaidEmployees)
		}

		empResult, err := f.Forecast(paidEmployeesValues, 24, config)
		if err != nil {
			f.logger.Error("Failed to forecast paid employees", "error", err)
		} else {
			results["paid_employees"] = empResult
		}
	}

	// Forecast revenue and expenses
	if len(historicalData.RevenueData) >= 3 {
		revenueValues := make([]float64, len(historicalData.RevenueData))
		expenseValues := make([]float64, len(historicalData.RevenueData))
		profitValues := make([]float64, len(historicalData.RevenueData))

		for i, data := range historicalData.RevenueData {
			revenueValues[i] = data.TotalRevenue
			expenseValues[i] = data.TotalExpenses
			profitValues[i] = data.Profit
		}

		// Revenue forecast
		revenueResult, err := f.Forecast(revenueValues, 24, config)
		if err != nil {
			f.logger.Error("Failed to forecast revenue", "error", err)
		} else {
			results["revenue"] = revenueResult
		}

		// Expenses forecast
		expenseResult, err := f.Forecast(expenseValues, 24, config)
		if err != nil {
			f.logger.Error("Failed to forecast expenses", "error", err)
		} else {
			results["expenses"] = expenseResult
		}

		// Profit forecast
		profitResult, err := f.Forecast(profitValues, 24, config)
		if err != nil {
			f.logger.Error("Failed to forecast profit", "error", err)
		} else {
			results["profit"] = profitResult
		}
	}

	// Forecast capital using payroll-based approach
	if len(historicalData.CapitalData) >= 3 && len(historicalData.WeeklyPayData) >= 1 {
		capitalResult := f.forecastCapitalRequirements(historicalData, weeklyPayResult)
		if capitalResult != nil {
			results["capital"] = capitalResult
		}
	}

	return results, nil
}

// forecastCapitalRequirements calculates capital needs based on payroll coverage
// This is more realistic than pure time-series forecasting for capital
func (f *HoltWintersForecaster) forecastCapitalRequirements(historicalData dto.HistoricalData, weeklyPayForecast *ForecastResult) *ForecastResult {
	// Get current capital and weekly pay
	currentCapital := historicalData.CapitalData[len(historicalData.CapitalData)-1].TotalCapital
	currentWeeklyPay := historicalData.WeeklyPayData[len(historicalData.WeeklyPayData)-1].TotalWeeklyPay

	// Calculate historical capital-to-payroll ratio (weeks of coverage)
	// This represents how many weeks of payroll the company keeps in reserve
	var coverageRatio float64
	if currentWeeklyPay > 0 {
		coverageRatio = currentCapital / currentWeeklyPay
	} else {
		coverageRatio = 4.0 // Default to 4 weeks coverage
	}

	// Cap coverage ratio to reasonable bounds (2-8 weeks is typical)
	if coverageRatio < 2.0 {
		coverageRatio = 2.0
	} else if coverageRatio > 8.0 {
		coverageRatio = 8.0
	}

	f.logger.Info("Capital coverage analysis",
		"current_capital", currentCapital,
		"current_weekly_pay", currentWeeklyPay,
		"coverage_weeks", coverageRatio)

	// If we have weekly pay forecasts, use them to project capital needs
	var capitalProjections []float64
	if weeklyPayForecast != nil && len(weeklyPayForecast.Values) > 0 {
		capitalProjections = make([]float64, len(weeklyPayForecast.Values))

		for i, projectedPay := range weeklyPayForecast.Values {
			// Capital needed = projected weekly pay × coverage ratio
			requiredCapital := projectedPay * coverageRatio

			// Capital should never decrease from current level (business constraint)
			if requiredCapital < currentCapital {
				requiredCapital = currentCapital
			}

			capitalProjections[i] = requiredCapital
		}
	} else {
		// Fallback: use simple growth based on historical capital growth rate
		capitalValues := make([]float64, len(historicalData.CapitalData))
		for i, data := range historicalData.CapitalData {
			capitalValues[i] = data.TotalCapital
		}

		config := DefaultForecastConfig()
		config.Phi = 0.90

		result, err := f.Forecast(capitalValues, 24, config)
		if err != nil {
			f.logger.Error("Failed to forecast capital with fallback", "error", err)
			return nil
		}

		// Ensure capital never drops below current
		capitalProjections = result.Values
		for i := range capitalProjections {
			if capitalProjections[i] < currentCapital {
				capitalProjections[i] = currentCapital
			}
		}
	}

	// Generate confidence intervals (±15% based on coverage ratio uncertainty)
	confidenceMin := make([]float64, len(capitalProjections))
	confidenceMax := make([]float64, len(capitalProjections))

	for i, cap := range capitalProjections {
		confidenceMin[i] = cap * 0.85
		confidenceMax[i] = cap * 1.15

		// Min should not go below current capital
		if confidenceMin[i] < currentCapital {
			confidenceMin[i] = currentCapital
		}
	}

	f.logger.Info("Capital forecast completed",
		"method", "payroll-coverage",
		"coverage_ratio", coverageRatio,
		"first_projection", capitalProjections[0],
		"last_projection", capitalProjections[len(capitalProjections)-1])

	return &ForecastResult{
		Values:        capitalProjections,
		ConfidenceMin: confidenceMin,
		ConfidenceMax: confidenceMax,
		Method:        "payroll-coverage",
	}
}

// ConvertToWeeklyProjections converts weekly forecasts to projection data
func (f *HoltWintersForecaster) ConvertToWeeklyProjections(forecasts map[string]*ForecastResult) dto.NewProjections {
	projections := dto.NewProjections{}

	extractWeekly := func(result *ForecastResult) (fourWeeks, twelveWeeks, twentyFourWeeks int64) {
		if result == nil || len(result.Values) < 24 {
			return 0, 0, 0
		}

		if len(result.Values) >= 4 {
			fourWeeks = int64(result.Values[3])
		}
		if len(result.Values) >= 12 {
			twelveWeeks = int64(result.Values[11])
		}
		if len(result.Values) >= 24 {
			twentyFourWeeks = int64(result.Values[23])
		}

		return fourWeeks, twelveWeeks, twentyFourWeeks
	}

	if empForecast, exists := forecasts["paid_employees"]; exists {
		fourW, twelveW, twentyFourW := extractWeekly(empForecast)
		projections.PaidEmployees = dto.DataProjection{
			ThirtyDays: fourW,
			SixtyDays:  twelveW,
			NinetyDays: twentyFourW,
		}
	}

	if payForecast, exists := forecasts["weekly_pay"]; exists {
		fourW, twelveW, twentyFourW := extractWeekly(payForecast)
		projections.WeeklyPay = dto.DataProjection{
			ThirtyDays: fourW,
			SixtyDays:  twelveW,
			NinetyDays: twentyFourW,
		}
	}

	if revenueForecast, exists := forecasts["revenue"]; exists {
		fourW, twelveW, twentyFourW := extractWeekly(revenueForecast)
		projections.Revenue = dto.DataProjection{
			ThirtyDays: fourW,
			SixtyDays:  twelveW,
			NinetyDays: twentyFourW,
		}
	}

	if profitForecast, exists := forecasts["profit"]; exists {
		fourW, twelveW, twentyFourW := extractWeekly(profitForecast)
		projections.Profit = dto.DataProjection{
			ThirtyDays: fourW,
			SixtyDays:  twelveW,
			NinetyDays: twentyFourW,
		}
	}

	if capitalForecast, exists := forecasts["capital"]; exists {
		fourW, twelveW, twentyFourW := extractWeekly(capitalForecast)
		projections.Capital = dto.DataProjection{
			ThirtyDays: fourW,
			SixtyDays:  twelveW,
			NinetyDays: twentyFourW,
		}
	}

	return projections
}

// CalculateAccuracyMetrics calculates accuracy metrics for the forecast
func (f *HoltWintersForecaster) CalculateAccuracyMetrics(actual, predicted []float64) map[string]float64 {
	if len(actual) == 0 || len(predicted) == 0 {
		return map[string]float64{}
	}

	n := min(len(actual), len(predicted))

	var sumAE, sumSquaredError, sumActual, sumAbsolutePercentage float64

	for i := 0; i < n; i++ {
		error := actual[i] - predicted[i]
		sumAE += math.Abs(error)
		sumSquaredError += error * error
		sumActual += actual[i]

		if actual[i] != 0 {
			sumAbsolutePercentage += math.Abs(error / actual[i])
		}
	}

	meanActual := sumActual / float64(n)
	metrics := map[string]float64{
		"mae":  sumAE / float64(n),
		"rmse": math.Sqrt(sumSquaredError / float64(n)),
	}

	if meanActual != 0 {
		metrics["mape"] = (sumAbsolutePercentage / float64(n)) * 100
	} else {
		metrics["mape"] = 0
	}

	return metrics
}

// Helper function for Go < 1.21 compatibility
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
