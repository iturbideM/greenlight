package httphelpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxBytes int64 = 1_048_576

// JSONDecode will try to decode json into pointer v. In case of unknown fields, they will be ignored
//
// If you do not wish to handle the error and are fine with 400 response on error
// feel free to use c.BindJSON(v)
func JSONDecode(c *gin.Context, v any) error {
	return jsonDecode(c, v, false)
}

// JSONDecode will try to decode json into pointer v. In case of unknown fields, an error will be returned
//
// If you do not wish to handle the error and are fine with 400 response on error
// feel free to use c.BindJSON(v)
func JSONDecodeNoUnknownFieldsAllowed(c *gin.Context, v any) error {
	return jsonDecode(c, v, false)
}

func jsonDecode(c *gin.Context, v any, allowUnknownFields bool) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)

	decoder := json.NewDecoder(c.Request.Body)
	if !allowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	err := decoder.Decode(v)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("request body contains badly-formed JSON (at position %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("request body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("request body contains an invalid value for the %q field", unmarshalTypeError.Field)
			}
			return fmt.Errorf("request body contains incorrect JSON type (at position %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("request body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("request body contains unknown field %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("request body must not be larger than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}

	}

	if e := decoder.Decode(&struct{}{}); e != io.EOF {
		err = errors.New("body must only contain a single JSON value")
		return err
	}

	return nil
}
