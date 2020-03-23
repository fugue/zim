package project

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fugue/zim/store"
)

// BufferedOutput is middleware that shows rule stdout and stderr
func BufferedOutput(runner Runner) Runner {
	return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
		buffer := &bytes.Buffer{}
		opts.Output = buffer
		opts.DebugOutput = buffer

		code, err := runner.Run(ctx, r, opts)

		output := strings.TrimSpace(buffer.String())
		if len(output) > 0 {
			for _, line := range strings.Split(output, "\n") {
				fmt.Println(line)
			}
		}
		return code, err
	})
}

// NewCapturedOutput is middleware that directs stdout and stderr to the given buffer
func NewCapturedOutput(w io.Writer) RunnerBuilder {
	return RunnerBuilder(func(runner Runner) Runner {
		return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
			opts.Output = w
			opts.DebugOutput = w
			return runner.Run(ctx, r, opts)
		})
	})
}

// NewArtifactUploader is middleware that uploads built artifacts
func NewArtifactUploader(s store.Store, dst string) RunnerBuilder {
	return RunnerBuilder(func(runner Runner) Runner {
		return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
			code, err := runner.Run(ctx, r, opts)
			if err != nil || code != OK {
				return code, err
			}
			for _, out := range r.Outputs() {
				dstKey := path.Join(dst, out.Name())
				fmt.Fprintln(opts.Output, "upl:", dstKey)
				if err := s.Put(ctx, dstKey, out.Path(), nil); err != nil {
					return Error, err
				}
			}
			return code, err
		})
	})
}

// NewArtifactDownloader is middleware that downloads built artifacts
func NewArtifactDownloader(s store.Store) RunnerBuilder {
	return RunnerBuilder(func(runner Runner) Runner {
		return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
			code, err := runner.Run(ctx, r, opts)
			if err != nil || code != OK {
				return code, err
			}
			for _, out := range r.Outputs() {
				srcKey := path.Join("artifacts", opts.BuildID, out.Name())
				fmt.Fprintln(opts.Output, "dnl:", srcKey, out.Name())
				if err := s.Get(ctx, srcKey, out.Path()); err != nil {
					return Error, err
				}
			}
			return code, err
		})
	})
}

// Logger is middleware that wraps logging around Rule execution
func Logger(runner Runner) Runner {
	return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {

		if opts.Output == nil {
			opts.Output = os.Stdout
		}

		fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()))
		startedAt := time.Now()

		code, err := runner.Run(ctx, r, opts)

		if code == UpToDate {
			fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()), Green("[CURRENT]"))
			return code, err
		} else if code == Cached {
			fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()), Green("[CACHED]"))
			return code, err
		}

		duration := time.Now().Sub(startedAt)
		durationStr := fmt.Sprintf("in %.3f sec", duration.Seconds())

		if err != nil {
			fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()),
				Bright(durationStr), Red("[FAILED]"))
		} else {
			fmt.Fprintln(opts.Output, "rule:", Bright(r.NodeID()),
				Bright(durationStr), Green("[OK]"))
		}
		return code, err
	})
}

// Debug is middleware that sets the debug flag to true
func Debug(runner Runner) Runner {
	return RunnerFunc(func(ctx context.Context, r *Rule, opts RunOpts) (Code, error) {
		opts.Debug = true
		return runner.Run(ctx, r, opts)
	})
}
