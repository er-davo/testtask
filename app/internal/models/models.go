package models

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var vld = validator.New()

func init() {
	// Register custom validation for MonthDate fields.
	vld.RegisterValidation("monthdate", func(fl validator.FieldLevel) bool {
		md, ok := fl.Field().Interface().(MonthDate)
		if !ok {
			return false
		}
		return !md.Time.IsZero()
	})
}

// Validate runs field validation based on struct tags.
func Validate(modelsStruct interface{}) error {
	return vld.Struct(modelsStruct)
}

// MonthDate represents a date limited to month and year precision.
// It is encoded and decoded in the "MM-YYYY" format.
type MonthDate struct {
	time.Time
}

// UnmarshalJSON parses a JSON string in "MM-YYYY" format.
func (m *MonthDate) UnmarshalJSON(b []byte) error {
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return fmt.Errorf("invalid month date: %w", err)
	}
	t, err := time.Parse("01-2006", s)
	if err != nil {
		return fmt.Errorf("invalid month date: %w", err)
	}
	m.Time = t
	return nil
}

// MarshalJSON formats MonthDate as "MM-YYYY".
func (m MonthDate) MarshalJSON() ([]byte, error) {
	s := fmt.Sprintf("%02d-%04d", m.Time.Month(), m.Time.Year())
	return []byte(strconv.Quote(s)), nil
}

// Subscription defines a user subscription entity.
type Subscription struct {
	ID          int64      `json:"id"`                                       // Subscription identifier.
	ServiceName string     `json:"service_name" validate:"required"`         // Service name.
	Price       int        `json:"price" validate:"gte=0"`                   // Monthly price.
	UserID      uuid.UUID  `json:"user_id" validate:"required"`              // Associated user ID.
	StartDate   MonthDate  `json:"start_date" validate:"required,monthdate"` // Start date (month-year).
	EndDate     *MonthDate `json:"end_date,omitempty"`                       // Optional end date.
}

// SummaryRequest defines the payload for requesting
// subscription cost summary within a given period.
type SummaryRequest struct {
	From        MonthDate `json:"from" validate:"required,monthdate"`           // Start of the period.
	To          MonthDate `json:"to" validate:"required,monthdate"`             // End of the period.
	UserID      *string   `json:"user_id,omitempty" validate:"omitempty,uuid4"` // Optional user filter.
	ServiceName *string   `json:"service_name,omitempty" validate:"omitempty"`  // Optional service filter.
}
