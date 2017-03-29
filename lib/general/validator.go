package general

import (
	"errors"
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"github.com/dzendmitry/logger"
)

type Validator struct {
	schemaLoaders map[string]gojsonschema.JSONLoader
	log logger.ILogger
}

func NewValidator(schemaLoaders map[string]gojsonschema.JSONLoader, log logger.ILogger) *Validator {
	return &Validator{schemaLoaders:schemaLoaders, log:log}
}

func (v *Validator) Validate(body []byte, validateLoaderName string) (error, []string) {
	sv, ok := v.schemaLoaders[validateLoaderName]
	if !ok {
		return errors.New(fmt.Sprintf("There is no %s schema loader", validateLoaderName)), nil
	}
	result, err := gojsonschema.Validate(sv, gojsonschema.NewBytesLoader(body))
	if err != nil {
		return errors.New(fmt.Sprintf("Json reg schema validation error: %s", err.Error())), nil
	}
	if !result.Valid() {
		errs := make([]string, 0, len(result.Errors()))
		for _, desc := range result.Errors() {
			if desc.Type() != "pattern" {
				errs = append(errs, desc.String())
			}
		}
		return errors.New("The document is not valid"), errs
	}
	return nil, nil
}