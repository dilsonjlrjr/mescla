package httpapi

import (
	"github.com/valyala/fasthttp"

	"paint-match-ai/api/service"
)

// Server é o servidor HTTP fasthttp completo, pronto pra ListenAndServe.
type Server struct {
	fast *fasthttp.Server
}

// NewServer monta o servidor com todas as rotas de svc. CORS liberado pra
// qualquer origem de propósito — app sem autenticação/sessão (ver
// stack-mescla-ai.md §5); em produção o nginx ainda serve front/ e api/ do
// mesmo domínio (reverse proxy em /api/), então CORS nem entra em jogo ali —
// isto cobre o dev direto (front rodando em outra porta) sem proxy.
func NewServer(svc *service.PaintService) *Server {
	r := NewRouter(svc)
	return &Server{
		fast: &fasthttp.Server{
			Handler: withCORS(r.Handler),
			Name:    "mescla-api",
		},
	}
}

// ListenAndServe bloqueia até o servidor cair ou dar erro.
func (s *Server) ListenAndServe(addr string) error {
	return s.fast.ListenAndServe(addr)
}

func withCORS(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
		ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")
		if string(ctx.Method()) == fasthttp.MethodOptions {
			ctx.SetStatusCode(fasthttp.StatusNoContent)
			return
		}
		next(ctx)
	}
}
