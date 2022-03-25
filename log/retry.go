package log

import (
	"time"

	"github.com/ydb-platform/ydb-go-sdk/v3/internal/errors"
	"github.com/ydb-platform/ydb-go-sdk/v3/retry"
	"github.com/ydb-platform/ydb-go-sdk/v3/trace"
)

func Retry(l Logger, details trace.Details) (t trace.Retry) {
	// nolint:nestif
	if details&trace.RetryEvents != 0 {
		l = l.WithName(`retry`)
		t.OnRetry = func(
			info trace.RetryLoopStartInfo,
		) func(
			trace.RetryLoopIntermediateInfo,
		) func(
			trace.RetryLoopDoneInfo,
		) {
			id := info.ID
			idempotent := info.Idempotent
			l.Tracef(`retry start {id:"%s",idempotent:%v}`, id, idempotent)
			start := time.Now()
			return func(info trace.RetryLoopIntermediateInfo) func(trace.RetryLoopDoneInfo) {
				if info.Error == nil {
					l.Tracef(`retry attempt done {id:"%s",latency:"%v"}`,
						id,
						time.Since(start),
					)
				} else {
					f := l.Errorf
					if !errors.IsYdb(info.Error) {
						f = l.Debugf
					}
					m := retry.Check(info.Error)
					f(`retry attempt failed {id:"%s",latency:"%v",error:"%s",retryable:%v,code:%d,deleteSession:%v}`,
						id,
						time.Since(start),
						info.Error,
						m.MustRetry(idempotent),
						m.StatusCode(),
						m.MustDeleteSession(),
					)
				}
				return func(info trace.RetryLoopDoneInfo) {
					if info.Error == nil {
						l.Tracef(`retry done {id:"%s",latency:"%v",attempts:%v}`,
							id,
							time.Since(start),
							info.Attempts,
						)
					} else {
						f := l.Errorf
						if !errors.IsYdb(info.Error) {
							f = l.Debugf
						}
						m := retry.Check(info.Error)
						f(`retry failed {id:"%s",latency:"%v",attempts:%v,error:"%s",retryable:%v,code:%d,deleteSession:%v}`,
							id,
							time.Since(start),
							info.Attempts,
							info.Error,
							m.MustRetry(idempotent),
							m.StatusCode(),
							m.MustDeleteSession(),
						)
					}
				}
			}
		}
	}
	return t
}