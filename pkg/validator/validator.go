package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type CustomValidator struct {
	Validator *validator.Validate
}

func NewCustomValidator() *CustomValidator {
	return &CustomValidator{
		Validator: validator.New(),
	}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.Validator.Struct(i); err != nil {
		if valErrs, ok := err.(validator.ValidationErrors); ok {
			var errMsgs []string
			for _, fe := range valErrs {
				errMsgs = append(errMsgs, fmt.Sprintf("field '%s' failed validation '%s'", fe.Field(), fe.Tag()))
			}
			return fmt.Errorf("validation error: %s", strings.Join(errMsgs, ", "))
		}
		return err
	}
	return nil
}

