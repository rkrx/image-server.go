// Package layoutscript runs operator-configured JavaScript composition layouts.
package layoutscript

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/dop251/goja"
)

//go:embed image_url.js
var imageURLSource string

var imageURLProgram = goja.MustCompile("image_url.js", imageURLSource, true)

type Script struct{ program *goja.Program }

type Result struct {
	Width  int
	Height int
	Canvas string
	Layout json.RawMessage
}

func Load(filename string) (*Script, error) {
	source, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	// Serialize inside JavaScript so the timeout also covers layout getters.
	const entrypoint = `
;if (typeof compose !== "function") throw new TypeError("expected function compose(url)");
(url => JSON.stringify(compose(url)));
`
	program, err := goja.Compile(filename, string(source)+entrypoint, true)
	if err != nil {
		return nil, err
	}
	script := &Script{program: program}
	_, cancel, err := script.start(context.Background())
	defer cancel()
	if err != nil {
		return nil, err
	}
	return script, nil
}

type invocation struct {
	vm      *goja.Runtime
	compose goja.Callable
}

func (s *Script) start(ctx context.Context) (invocation, func(), error) {
	vm := goja.New()
	vm.SetMaxCallStackSize(512)
	ctx, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
	stop := context.AfterFunc(ctx, func() { vm.Interrupt(ctx.Err()) })
	finish := func() { stop(); cancel() }
	if _, err := vm.RunProgram(imageURLProgram); err != nil {
		return invocation{}, finish, err
	}
	value, err := vm.RunProgram(s.program)
	if err != nil {
		return invocation{}, finish, err
	}
	compose, ok := goja.AssertFunction(value)
	if !ok {
		return invocation{}, finish, errors.New("expected function compose(url)")
	}
	return invocation{vm, compose}, finish, nil
}

func (s *Script) Run(ctx context.Context, pathname string, values url.Values) (Result, error) {
	var result Result
	params := make(map[string]string, len(values))
	for name, entries := range values {
		if len(entries) != 1 {
			return result, errors.New("duplicate script parameter")
		}
		if name != "hash" {
			params[name] = entries[0]
		}
	}
	call, cancel, err := s.start(ctx)
	defer cancel()
	if err != nil {
		return result, err
	}
	input := call.vm.ToValue(map[string]any{"pathname": pathname, "params": params})
	value, err := call.compose(goja.Undefined(), input)
	if err != nil {
		return result, err
	}
	if goja.IsUndefined(value) {
		return result, errors.New("compose must return a layout")
	}
	decoder := json.NewDecoder(strings.NewReader(value.String()))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&result)
	return result, err
}
