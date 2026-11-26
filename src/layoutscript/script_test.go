package layoutscript

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"image-server/src/imaging"

	"github.com/stretchr/testify/require"
)

func TestOverlayUsesRequestImageAsBackground(t *testing.T) {
	script, err := Load("../../doc/examples/overlay.js")
	require.NoError(t, err)
	result, err := script.Run(t.Context(), "/landscape.jpg", nil)
	require.NoError(t, err)
	require.Equal(t, 640, result.Width)
	require.Equal(t, 480, result.Height)
	require.Empty(t, result.Canvas)
	nodes, err := imaging.ParseComposition(string(result.Layout), imaging.CompositionLimits{MaxNodes: 1, MaxImages: 1, MaxDepth: 1})
	require.NoError(t, err)
	require.Len(t, nodes, 1)
	require.Equal(t, "/markers/corner.png", nodes[0].URL)
}

func TestScriptErrorsAndIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.js")
	for _, source := range []string{"function compose(", "const other = 0"} {
		require.NoError(t, os.WriteFile(path, []byte(source), 0600))
		_, err := Load(path)
		require.Error(t, err)
	}
	require.NoError(t, os.WriteFile(path, []byte(`
        let count = 0;
        function compose(url) {
            switch (url.params.mode) {
            case "loop": for (;;) {}
            case "getter": return {get width() { for (;;) {} }};
            case "empty": return;
            }
            return {width: ++count, height:1, layout:[]};
        }`), 0600))
	script, err := Load(path)
	require.NoError(t, err)
	for _, mode := range []string{"loop", "getter", "empty"} {
		_, err := script.Run(t.Context(), "/", url.Values{"mode": {mode}})
		require.Error(t, err)
	}
	for i := 0; i < 2; i++ {
		result, err := script.Run(t.Context(), "/", nil)
		require.NoError(t, err)
		require.Equal(t, 1, result.Width)
	}
}
