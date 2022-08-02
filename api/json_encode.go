package api

import (
	"encoding/json"
	"go.uber.org/zap"
	"io"
)

func writeJSON(w io.Writer, data any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "    ")

	err := enc.Encode(data)
	if err != nil {
		zap.L().Error("unable to writeJSON", zap.Error(err))
	}
}
