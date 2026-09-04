// Package httpapi expõe api/service.PaintService por HTTP (fasthttp) — o
// contrato que front/ consome pela rede. wails/ NUNCA passa por aqui: continua
// importando api/service direto (bind Wails, offline, sem HTTP).
package httpapi

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
)

func writeJSON(ctx *fasthttp.RequestCtx, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "serializando resposta: "+err.Error())
		return
	}
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json; charset=utf-8")
	ctx.SetBody(body)
}

type errorBody struct {
	Error string `json:"error"`
}

func writeError(ctx *fasthttp.RequestCtx, status int, message string) {
	writeJSON(ctx, status, errorBody{Error: message})
}

// readJSON decodifica o corpo da requisição em v. Em erro, já escreve a
// resposta 400 e devolve false — o handler só precisa `if !readJSON(...) { return }`.
func readJSON(ctx *fasthttp.RequestCtx, v any) bool {
	if err := json.Unmarshal(ctx.PostBody(), v); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "corpo inválido: "+err.Error())
		return false
	}
	return true
}

func pathInt64(ctx *fasthttp.RequestCtx, name string) (int64, bool) {
	raw, _ := ctx.UserValue(name).(string)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, name+" inválido: "+raw)
		return 0, false
	}
	return id, true
}

func queryInt64(ctx *fasthttp.RequestCtx, name string, def int64) int64 {
	raw := string(ctx.QueryArgs().Peek(name))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func queryInt(ctx *fasthttp.RequestCtx, name string, def int) int {
	return int(queryInt64(ctx, name, int64(def)))
}

func queryFloat64(ctx *fasthttp.RequestCtx, name string, def float64) float64 {
	raw := string(ctx.QueryArgs().Peek(name))
	if raw == "" {
		return def
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return def
	}
	return v
}

func queryUint8(ctx *fasthttp.RequestCtx, name string) uint8 {
	v := queryInt64(ctx, name, 0)
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// queryInt64List lê algo como "?ids=1,2,3" — usado por /compare.
func queryInt64List(ctx *fasthttp.RequestCtx, name string) []int64 {
	raw := string(ctx.QueryArgs().Peek(name))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if v, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}
