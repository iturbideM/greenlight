package httphelpers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"greenlight/pkg/validator"

	"github.com/gin-gonic/gin"
)

func ReadIDParam(c *gin.Context) (int64, error) {
	params := c.Params

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}

type Envelope map[string]any

func WriteJson(c *gin.Context, status int, data Envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		c.Writer.Header()[key] = value
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(status)
	c.Writer.Write(js)

	return nil
}

func ReadString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}

func ReadCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue
	}

	return strings.Split(csv, ",")
}

func ReadInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer")
		return defaultValue
	}

	return i
}
