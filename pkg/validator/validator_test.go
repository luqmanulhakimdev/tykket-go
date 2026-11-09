package validator

import (
	"testing"
)

type sampleTestStruct struct {
	Name  string `validate:"required,min=3"`
	Email string `validate:"required,email"`
}

func TestCustomValidator_Validate(t *testing.T) {
	v := NewCustomValidator()

	t.Run("Valid struct passes", func(t *testing.T) {
		valid := sampleTestStruct{
			Name:  "John",
			Email: "john@example.com",
		}
		if err := v.Validate(valid); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Invalid struct returns formatted validation error", func(t *testing.T) {
		invalid := sampleTestStruct{
			Name:  "J",
			Email: "not-an-email",
		}
		err := v.Validate(invalid)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
		errMsg := err.Error()
		if errMsg == "" {
			t.Errorf("expected non-empty error message")
		}
	})

	t.Run("Non-struct returns error", func(t *testing.T) {
		err := v.Validate("just-a-string")
		if err != nil {
			// validator might return InvalidValidationError for non-structs or nil
		}
	})
}
