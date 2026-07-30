package ratelimit

import (
	"log/slog"
	"net/http"

	"github.com/go-redis/redis_rate/v10"
)

// Middleware создаёт HTTP middleware с распределённым rate limiter на основе redis_rate.
//
// Библиотека redis_rate использует алгоритм GCRA (Generic Cell Rate Algorithm) —
// это вариант token bucket, реализованный атомарно в Redis через Lua-скрипт.
// Благодаря Redis все инстансы сервиса разделяют общий счётчик запросов,
// поэтому лимит работает глобально, а не per-instance.
//
// rate  — сколько запросов в секунду разрешено в среднем.
// burst — максимальный размер всплеска (сколько запросов можно сделать одновременно).
func Middleware(limiter *redis_rate.Limiter, rate, burst int) func(http.Handler) http.Handler {
	limit := redis_rate.Limit{
		Rate:   rate,
		Burst:  burst,
		Period: redis_rate.PerSecond(rate).Period,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Библиотека redis_rate сама добавляет префикс "rate:" к ключу,
			// поэтому дополнительный префикс не нужен.
			key := r.URL.Path

			res, err := limiter.Allow(r.Context(), key, limit)
			if err != nil {
				slog.WarnContext(
					r.Context(), "rate limiter недоступен, пропускаем запрос",
					"error", err,
					"path", r.URL.Path,
				)
				next.ServeHTTP(w, r)
				return
			}

			if res.Allowed == 0 {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)

				if _, wErr := w.Write([]byte("превышен лимит запросов")); wErr != nil {
					slog.WarnContext(r.Context(), "не удалось записать тело 429-ответа", "error", wErr)
				}

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
