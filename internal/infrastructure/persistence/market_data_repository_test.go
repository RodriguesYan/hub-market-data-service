package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/RodriguesYan/hub-market-data-service/internal/application/dto"
	"github.com/RodriguesYan/hub-market-data-service/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDatabase is a mock implementation of the database interface
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) Query(query string, args ...interface{}) (database.Rows, error) {
	callArgs := m.Called(query, args)
	return callArgs.Get(0).(database.Rows), callArgs.Error(1)
}

func (m *MockDatabase) QueryContext(ctx context.Context, query string, args ...interface{}) (database.Rows, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(database.Rows), callArgs.Error(1)
}

func (m *MockDatabase) QueryRow(query string, args ...interface{}) database.Row {
	callArgs := m.Called(query, args)
	return callArgs.Get(0).(database.Row)
}

func (m *MockDatabase) QueryRowContext(ctx context.Context, query string, args ...interface{}) database.Row {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(database.Row)
}

func (m *MockDatabase) Exec(query string, args ...interface{}) (database.Result, error) {
	callArgs := m.Called(query, args)
	return callArgs.Get(0).(database.Result), callArgs.Error(1)
}

func (m *MockDatabase) ExecContext(ctx context.Context, query string, args ...interface{}) (database.Result, error) {
	callArgs := m.Called(ctx, query, args)
	return callArgs.Get(0).(database.Result), callArgs.Error(1)
}

func (m *MockDatabase) Begin() (database.Transaction, error) {
	callArgs := m.Called()
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(database.Transaction), callArgs.Error(1)
}

func (m *MockDatabase) BeginTx(ctx context.Context, opts *sql.TxOptions) (database.Transaction, error) {
	callArgs := m.Called(ctx, opts)
	if callArgs.Get(0) == nil {
		return nil, callArgs.Error(1)
	}
	return callArgs.Get(0).(database.Transaction), callArgs.Error(1)
}

func (m *MockDatabase) Get(dest interface{}, query string, args ...interface{}) error {
	callArgs := m.Called(dest, query, args)
	return callArgs.Error(0)
}

func (m *MockDatabase) Select(dest interface{}, query string, args ...interface{}) error {
	callArgs := m.Called(dest, query, args)

	// If there's data to return, copy it to dest
	if len(callArgs) > 1 {
		if dtos, ok := callArgs.Get(1).([]dto.MarketDataDTO); ok {
			destSlice := dest.(*[]dto.MarketDataDTO)
			*destSlice = dtos
		}
	}

	return callArgs.Error(0)
}

func (m *MockDatabase) Ping() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

func (m *MockDatabase) Close() error {
	callArgs := m.Called()
	return callArgs.Error(0)
}

func TestNewMarketDataRepository(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}

	// Act
	repo := NewMarketDataRepository(mockDB)

	// Assert
	assert.NotNil(t, repo)
	assert.IsType(t, &MarketDataRepository{}, repo)
}

func TestMarketDataRepository_GetMarketData_Success(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	expectedDTOs := []dto.MarketDataDTO{
		{Id: 1, Symbol: "AAPL", Name: "Apple Inc.", LastQuote: 155.50, Category: 1},
		{Id: 2, Symbol: "GOOGL", Name: "Alphabet Inc.", LastQuote: 2650.75, Category: 1},
		{Id: 3, Symbol: "MSFT", Name: "Microsoft Corporation", LastQuote: 285.25, Category: 1},
	}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1,$2,$3)"
	expectedArgs := []interface{}{"AAPL", "GOOGL", "MSFT"}

	// Mock successful database query
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result))

	// Verify the domain models are correctly mapped
	assert.Equal(t, "AAPL", result[0].Symbol)
	assert.Equal(t, "Apple Inc.", result[0].Name)
	assert.Equal(t, float32(155.50), result[0].LastQuote)
	assert.Equal(t, 1, result[0].Category)

	assert.Equal(t, "GOOGL", result[1].Symbol)
	assert.Equal(t, "MSFT", result[2].Symbol)
}

func TestMarketDataRepository_GetMarketData_SingleSymbol(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"AAPL"}
	expectedDTOs := []dto.MarketDataDTO{
		{Id: 1, Symbol: "AAPL", Name: "Apple Inc.", LastQuote: 155.50, Category: 1},
	}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1)"
	expectedArgs := []interface{}{"AAPL"}

	// Mock successful database query
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result))
	assert.Equal(t, "AAPL", result[0].Symbol)
	assert.Equal(t, "Apple Inc.", result[0].Name)
	assert.Equal(t, float32(155.50), result[0].LastQuote)
	assert.Equal(t, 1, result[0].Category)
}

func TestMarketDataRepository_GetMarketData_EmptySymbols(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{}
	expectedDTOs := []dto.MarketDataDTO{}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ()"
	expectedArgs := []interface{}{}

	// Mock successful database query with empty result
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}

func TestMarketDataRepository_GetMarketData_DatabaseError(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"AAPL", "GOOGL"}
	databaseError := errors.New("connection lost")

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1,$2)"
	expectedArgs := []interface{}{"AAPL", "GOOGL"}

	// Mock database error
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(databaseError)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch market data")
	assert.Contains(t, err.Error(), "connection lost")
	assert.Contains(t, err.Error(), "[AAPL GOOGL]")
}

func TestMarketDataRepository_GetMarketData_NoDataFound(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"INVALID", "NOTFOUND"}
	expectedDTOs := []dto.MarketDataDTO{} // Empty result

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1,$2)"
	expectedArgs := []interface{}{"INVALID", "NOTFOUND"}

	// Mock successful query but no data found
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}

func TestMarketDataRepository_GetMarketData_PartialDataFound(t *testing.T) {
	// Arrange - Request 3 symbols but only 2 found
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"AAPL", "INVALID", "GOOGL"}
	expectedDTOs := []dto.MarketDataDTO{
		{Id: 1, Symbol: "AAPL", Name: "Apple Inc.", LastQuote: 155.50, Category: 1},
		{Id: 2, Symbol: "GOOGL", Name: "Alphabet Inc.", LastQuote: 2650.75, Category: 1},
	}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1,$2,$3)"
	expectedArgs := []interface{}{"AAPL", "INVALID", "GOOGL"}

	// Mock successful query with partial data
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result)) // Only 2 out of 3 symbols found
	assert.Equal(t, "AAPL", result[0].Symbol)
	assert.Equal(t, "GOOGL", result[1].Symbol)
}

func TestMarketDataRepository_GetMarketData_DifferentCategories(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"AAPL", "VOO", "BTC"}
	expectedDTOs := []dto.MarketDataDTO{
		{Id: 1, Symbol: "AAPL", Name: "Apple Inc.", LastQuote: 155.50, Category: 1},          // Stock
		{Id: 2, Symbol: "VOO", Name: "Vanguard S&P 500 ETF", LastQuote: 385.25, Category: 2}, // ETF
		{Id: 3, Symbol: "BTC", Name: "Bitcoin", LastQuote: 45000.00, Category: 3},            // Crypto
	}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1,$2,$3)"
	expectedArgs := []interface{}{"AAPL", "VOO", "BTC"}

	// Mock successful database query
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result))

	// Verify different categories
	assert.Equal(t, 1, result[0].Category) // Stock
	assert.Equal(t, 2, result[1].Category) // ETF
	assert.Equal(t, 3, result[2].Category) // Crypto
}

func TestMarketDataRepository_GetMarketData_LargeSymbolList(t *testing.T) {
	// Arrange - Test with many symbols
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	// Create 50 symbols
	symbols := make([]string, 50)
	expectedDTOs := make([]dto.MarketDataDTO, 50)
	expectedArgs := make([]interface{}, 50)

	for i := 0; i < 50; i++ {
		symbol := "SYM" + fmt.Sprintf("%02d", i)
		symbols[i] = symbol
		expectedArgs[i] = symbol
		expectedDTOs[i] = dto.MarketDataDTO{
			Id:        i + 1,
			Symbol:    symbol,
			Name:      "Company " + symbol,
			LastQuote: float32(100.0 + float64(i)),
			Category:  1,
		}
	}

	// Build expected query with 50 placeholders
	placeholders := make([]string, 50)
	for i := 0; i < 50; i++ {
		placeholders[i] = "$" + fmt.Sprintf("%d", i+1)
	}
	expectedQuery := "SELECT * FROM market_data WHERE symbol IN (" + strings.Join(placeholders, ",") + ")"

	// Mock successful database query
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 50, len(result))

	// Verify first and last items
	assert.Equal(t, "SYM00", result[0].Symbol)
	assert.Equal(t, "SYM49", result[49].Symbol)
}

func TestMarketDataRepository_GetMarketData_SpecialCharacters(t *testing.T) {
	// Arrange - Test symbols with special characters
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"BRK.B", "BRK.A", "SPY"}
	expectedDTOs := []dto.MarketDataDTO{
		{Id: 1, Symbol: "BRK.B", Name: "Berkshire Hathaway Inc. Class B", LastQuote: 275.50, Category: 1},
		{Id: 2, Symbol: "BRK.A", Name: "Berkshire Hathaway Inc. Class A", LastQuote: 415000.00, Category: 1},
		{Id: 3, Symbol: "SPY", Name: "SPDR S&P 500 ETF Trust", LastQuote: 420.75, Category: 2},
	}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1,$2,$3)"
	expectedArgs := []interface{}{"BRK.B", "BRK.A", "SPY"}

	// Mock successful database query
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result))
	assert.Equal(t, "BRK.B", result[0].Symbol)
	assert.Equal(t, "BRK.A", result[1].Symbol)
}

func TestMarketDataRepository_GetMarketData_QueryGeneration(t *testing.T) {
	// Arrange - Test that the SQL query is generated correctly for different input sizes
	testCases := []struct {
		name          string
		symbols       []string
		expectedQuery string
		expectedArgs  []interface{}
	}{
		{
			name:          "single symbol",
			symbols:       []string{"AAPL"},
			expectedQuery: "SELECT * FROM market_data WHERE symbol IN ($1)",
			expectedArgs:  []interface{}{"AAPL"},
		},
		{
			name:          "two symbols",
			symbols:       []string{"AAPL", "GOOGL"},
			expectedQuery: "SELECT * FROM market_data WHERE symbol IN ($1,$2)",
			expectedArgs:  []interface{}{"AAPL", "GOOGL"},
		},
		{
			name:          "three symbols",
			symbols:       []string{"AAPL", "GOOGL", "MSFT"},
			expectedQuery: "SELECT * FROM market_data WHERE symbol IN ($1,$2,$3)",
			expectedArgs:  []interface{}{"AAPL", "GOOGL", "MSFT"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &MockDatabase{}
			defer mockDB.AssertExpectations(t)

			expectedDTOs := make([]dto.MarketDataDTO, len(tc.symbols))
			for i, symbol := range tc.symbols {
				expectedDTOs[i] = dto.MarketDataDTO{
					Id:        i + 1,
					Symbol:    symbol,
					Name:      "Company " + symbol,
					LastQuote: float32(100.0 + float64(i)),
					Category:  1,
				}
			}

			// Mock database call with exact expected query and args
			mockDB.On("Select",
				mock.AnythingOfType("*[]dto.MarketDataDTO"),
				tc.expectedQuery,
				tc.expectedArgs,
			).Return(nil, expectedDTOs)

			repo := NewMarketDataRepository(mockDB)

			// Act
			result, err := repo.GetMarketData(tc.symbols)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, len(tc.symbols), len(result))
		})
	}
}

func TestMarketDataRepository_GetMarketData_DataMapping(t *testing.T) {
	// Arrange - Test that DTO to Domain mapping works correctly
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	symbols := []string{"TEST"}
	expectedDTOs := []dto.MarketDataDTO{
		{
			Id:        123,
			Symbol:    "TEST",
			Name:      "Test Company Inc.",
			LastQuote: 99.99,
			Category:  2,
		},
	}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ($1)"
	expectedArgs := []interface{}{"TEST"}

	// Mock successful database query
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result))

	// Verify all fields are correctly mapped from DTO to domain model
	domainModel := result[0]
	assert.Equal(t, "TEST", domainModel.Symbol)
	assert.Equal(t, "Test Company Inc.", domainModel.Name)
	assert.Equal(t, float32(99.99), domainModel.LastQuote)
	assert.Equal(t, 2, domainModel.Category)

	// Note: ID field is not mapped to domain model as it's not part of MarketDataModel
}

func TestMarketDataRepository_GetMarketData_NilSymbols(t *testing.T) {
	// Arrange
	mockDB := &MockDatabase{}
	defer mockDB.AssertExpectations(t)

	var symbols []string = nil
	expectedDTOs := []dto.MarketDataDTO{}

	expectedQuery := "SELECT * FROM market_data WHERE symbol IN ()"
	expectedArgs := []interface{}{}

	// Mock successful database query
	mockDB.On("Select",
		mock.AnythingOfType("*[]dto.MarketDataDTO"),
		expectedQuery,
		expectedArgs,
	).Return(nil, expectedDTOs)

	repo := NewMarketDataRepository(mockDB)

	// Act
	result, err := repo.GetMarketData(symbols)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}
