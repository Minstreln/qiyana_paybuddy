package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"qiyana_paybuddy/pkg/utils"
	"reflect"
	"regexp"
	"strings"
)

func CheckFieldNames(model interface{}) []string {
	val := reflect.TypeOf(model)
	fields := []string{}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldToAdd := strings.TrimSuffix(field.Tag.Get("json"), ",omitempty")
		fields = append(fields, fieldToAdd) // get json tag
	}
	return fields
}

func CheckBlankFields(value interface{}) error {
	val := reflect.ValueOf(value)
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() == reflect.String && field.String() == "" {
			// http.Error(w, "All fields are required", http.StatusBadRequest)
			return utils.ErrorHandler(errors.New("all fields are required"), "all fields are required")
		}
	}
	return nil
}

func ValidatePhoneNumber(w http.ResponseWriter, phone string) bool {
	if !regexp.MustCompile(`^\+234[0-9]{10}$`).MatchString(phone) {
		utils.WriteError(w, "invalid phone number format", http.StatusBadRequest)
		return false
	}

	return true
}

func ValidateProvider(provider string) error {
	validProviders := map[string]bool{"MTN": true, "AIRTEL": true, "GLO": true, "9MOBILE": true}
	if !validProviders[provider] {
		return fmt.Errorf("invalid provider")
	}
	return nil
}
