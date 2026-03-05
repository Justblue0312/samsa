package validate

import (
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
)

func New() *validator.Validate {
	return validator.New()
}

func InitValidator(validate *validator.Validate) {
	validate.RegisterValidation("hhmm", validateHHMM)
	validate.RegisterValidation("timezone", validateTimezone)
}

// validateHHMM checks if the string is in the format "HH:MM" and represents a valid time.
func validateHHMM(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	re := regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)
	return re.MatchString(value)
}

// validateTimezone checks if the string is a valid IANA timezone.
func validateTimezone(fl validator.FieldLevel) bool {
	tz := fl.Field().String()
	_, err := time.LoadLocation(tz)
	return err == nil
}
