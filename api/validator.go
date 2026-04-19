package api

import (
	"simplebank/util"

	"github.com/go-playground/validator/v10"
)

// 自定义验证器
var validCurrency validator.Func = func(fl validator.FieldLevel) bool {
	if currency, ok := fl.Field().Interface().(string); ok {
		return util.IsSupportedCurrency(currency)
	}
	return false
}
