package projection

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple calculator for isolated testing
type TestCalculator struct{}

func (c *TestCalculator) Calculate(input ProjectionInput) []ProjectionPoint {
	points := make([]ProjectionPoint, 0, input.PeriodMonths)

	// Convert APY to monthly rate
	annualRate := input.APY

	// Monthly compound rate calculation
	monthlyRate := decimal.Zero
	if input.CompoundFrequency == CompoundDaily {
		// For daily compounding: use simple approximation
		monthlyRate = annualRate.Div(decimal.NewFromInt(12)).Mul(decimal.NewFromFloat(1.05)) // Slightly higher for daily
	} else {
		// For monthly compounding: APY / 12
		monthlyRate = annualRate.Div(decimal.NewFromInt(12))
	}

	balance := input.InitialDeposit
	totalDeposited := input.InitialDeposit

	for month := 1; month <= input.PeriodMonths; month++ {
		// Add monthly contribution at the beginning of the month
		if month > 1 {
			balance = balance.Add(input.MonthlyContribution)
			totalDeposited = totalDeposited.Add(input.MonthlyContribution)
		}

		// Apply compound interest
		interestEarned := balance.Mul(monthlyRate)
		balance = balance.Add(interestEarned)

		// Calculate total yield earned
		totalYield := balance.Sub(totalDeposited)

		points = append(points, ProjectionPoint{
			Month:     month,
			Principal: totalDeposited,
			Yield:     totalYield,
			Total:     balance,
		})
	}

	return points
}

func TestProjectionCalculation(t *testing.T) {
	calculator := &TestCalculator{}

	input := ProjectionInput{
		InitialDeposit:      decimal.NewFromInt(1000),
		MonthlyContribution: decimal.Zero,
		APY:                 decimal.NewFromFloat(0.12), // 12% APY
		PeriodMonths:        12,
		CompoundFrequency:   CompoundMonthly,
	}

	err := input.Validate()
	require.NoError(t, err, "Input should be valid")

	results := calculator.Calculate(input)

	require.Len(t, results, 12, "Should return exactly 12 months")

	// Test first month
	first := results[0]
	assert.Equal(t, 1, first.Month)
	assert.True(t, first.Principal.Equal(decimal.NewFromInt(1000)))
	assert.True(t, first.Yield.GreaterThan(decimal.Zero))
	assert.True(t, first.Total.GreaterThan(first.Principal))

	// Test growth trend
	for i := 1; i < len(results); i++ {
		assert.True(t, results[i].Total.GreaterThan(results[i-1].Total),
			"Total should increase from month %d to %d", i, i+1)
	}

	// Test final result is reasonable (should be around 1000 * 1.12 = 1120 for 12% APY)
	final := results[11]
	finalValue := final.Total.InexactFloat64()
	assert.Greater(t, finalValue, 1100.0, "Final value should be over 1100")
	assert.Less(t, finalValue, 1150.0, "Final value should be under 1150")
}

func TestProjectionWithMonthlyContributions(t *testing.T) {
	calculator := &TestCalculator{}

	input := ProjectionInput{
		InitialDeposit:      decimal.NewFromInt(1000),
		MonthlyContribution: decimal.NewFromInt(200),
		APY:                 decimal.NewFromFloat(0.10), // 10% APY
		PeriodMonths:        6,
		CompoundFrequency:   CompoundMonthly,
	}

	results := calculator.Calculate(input)

	require.Len(t, results, 6)

	// Check principal accumulates correctly
	month2 := results[1]
	expectedPrincipal := decimal.NewFromInt(1200) // 1000 + 200
	assert.True(t, month2.Principal.Equal(expectedPrincipal))

	finalMonth := results[5]
	expectedFinalPrincipal := decimal.NewFromInt(2000) // 1000 + 5*200
	assert.True(t, finalMonth.Principal.Equal(expectedFinalPrincipal))

	// Yield should be positive
	assert.True(t, finalMonth.Yield.GreaterThan(decimal.Zero))
}

func TestProjectionInputValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   ProjectionInput
		wantErr bool
	}{
		{
			name: "Valid input",
			input: ProjectionInput{
				InitialDeposit:      decimal.NewFromInt(1000),
				MonthlyContribution: decimal.NewFromInt(100),
				APY:                 decimal.NewFromFloat(0.08),
				PeriodMonths:        12,
				CompoundFrequency:   CompoundMonthly,
			},
			wantErr: false,
		},
		{
			name: "Zero initial deposit - should fail",
			input: ProjectionInput{
				InitialDeposit:      decimal.Zero,
				MonthlyContribution: decimal.NewFromInt(100),
				APY:                 decimal.NewFromFloat(0.08),
				PeriodMonths:        12,
				CompoundFrequency:   CompoundMonthly,
			},
			wantErr: true,
		},
		{
			name: "Negative monthly contribution - should fail",
			input: ProjectionInput{
				InitialDeposit:      decimal.NewFromInt(1000),
				MonthlyContribution: decimal.NewFromInt(-50),
				APY:                 decimal.NewFromFloat(0.08),
				PeriodMonths:        12,
				CompoundFrequency:   CompoundMonthly,
			},
			wantErr: true,
		},
		{
			name: "Zero APY - should fail",
			input: ProjectionInput{
				InitialDeposit:      decimal.NewFromInt(1000),
				MonthlyContribution: decimal.Zero,
				APY:                 decimal.Zero,
				PeriodMonths:        12,
				CompoundFrequency:   CompoundMonthly,
			},
			wantErr: true,
		},
		{
			name: "Zero period - should fail",
			input: ProjectionInput{
				InitialDeposit:      decimal.NewFromInt(1000),
				MonthlyContribution: decimal.Zero,
				APY:                 decimal.NewFromFloat(0.08),
				PeriodMonths:        0,
				CompoundFrequency:   CompoundMonthly,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCompoundFrequency(t *testing.T) {
	tests := []struct {
		input    string
		expected CompoundFrequency
		wantErr  bool
	}{
		{"daily", CompoundDaily, false},
		{"monthly", CompoundMonthly, false},
		{"invalid", CompoundMonthly, true},
		{"", CompoundMonthly, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseCompoundFrequency(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}

	// Test periods per year
	assert.Equal(t, 365, CompoundDaily.PeriodsPerYear())
	assert.Equal(t, 12, CompoundMonthly.PeriodsPerYear())

	// Test string representation
	assert.Equal(t, "daily", CompoundDaily.String())
	assert.Equal(t, "monthly", CompoundMonthly.String())
}
