package agent

import (
	"bytes"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/testenv"
)

func grokBotArgs(t *testing.T, cmd string) []string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX command runs on Grok Bot's cloud computer")
	}
	_, tail, ok := strings.Cut(cmd, " && node ")
	if !ok {
		t.Fatalf("missing node command: %q", cmd)
	}
	out, err := exec.Command("/bin/sh", "-c", `node() { printf '%s\0' "$@"; }; node `+tail).CombinedOutput()
	if err != nil {
		t.Fatalf("shell: %v: %s", err, out)
	}
	parts := bytes.Split(bytes.TrimSuffix(out, []byte{0}), []byte{0})
	args := make([]string, len(parts))
	for i, p := range parts {
		args[i] = string(p)
	}
	return args
}
func grokBotArg(args []string, flag string) string {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestGrokBotFreshCloudAgentAndConfiguration(t *testing.T) {
	home := t.TempDir()
	testenv.SetHome(t, home)
	a := grokBot(home)
	if !a.Detected() {
		t.Fatal("cloud agent hidden without a local installation")
	}
	if a.Launch() != "" {
		t.Fatal("unconfigured agent produces an unusable command")
	}
	if err := provider.Save(provider.Provider{ID: "vendor", Name: "Vendor", Chat: "https://api.example/v1", Key: "synthetic", Models: []string{"model"}}); err != nil {
		t.Fatal(err)
	}
	if err := a.Field("model").Set("magpie/vendor/model"); err != nil {
		t.Fatal(err)
	}
	if a.Launch() == "" {
		t.Fatal(a.Notice())
	}
	if err := a.Field("mode").Set("gateway"); err != nil {
		t.Fatal(err)
	}
	if a.Launch() != "" {
		t.Fatal("gateway command without cloud URL")
	}
	f := a.Field("gateway-url")
	if f == nil {
		t.Fatal("missing gateway URL field")
	}
	for _, u := range []string{"http://127.0.0.1:3425/v1", "http://localhost/v1", "http://[::1]/v1", "file:///etc/passwd"} {
		if err := f.Set(u); err == nil {
			t.Fatalf("accepted unreachable URL %s", u)
		}
	}
	if err := f.Set("https://gateway.example/v1"); err != nil {
		t.Fatal(err)
	}
	if a.Launch() != "" {
		t.Fatal("gateway command without shared key")
	}
	if err := access.ConfigureLAN(true, false); err != nil {
		t.Fatal(err)
	}
	prompt := a.Launch()
	if prompt == "" {
		t.Fatal(a.Notice())
	}
	args := grokBotArgs(t, strings.SplitN(prompt, "\n\n", 2)[1])
	if got := grokBotArg(args, "--key"); got != access.LANSecret() {
		t.Fatal("cloud command uses a loopback-only key")
	}
	if got := grokBotArg(args, "--model"); got != "vendor/model" {
		t.Fatalf("lost provider: %s", got)
	}
	if err := a.Field("mode").Set("typo"); err == nil {
		t.Fatal("invalid mode accepted")
	}
}

func TestGrokBotGatewayPreservesProviderAndGroup(t *testing.T) {
	testenv.SetHome(t, t.TempDir())
	for _, id := range []string{"aaa", "zzz"} {
		if err := provider.Save(provider.Provider{ID: id, Name: id, Chat: "https://api.example/v1", Key: "synthetic", Models: []string{"shared"}}); err != nil {
			t.Fatal(err)
		}
	}
	for _, model := range []string{"zzz/shared", provider.GroupPrefix + "fallback"} {
		cmd := GrokBotLaunchCommand("magpie/"+model, "gateway", "https://gateway.example/v1", "synthetic")
		if got := grokBotArg(grokBotArgs(t, cmd), "--model"); got != model {
			t.Fatalf("requested %q, got %q", model, got)
		}
	}
}

func TestGrokBotShellArgumentIntegrity(t *testing.T) {
	testenv.SetHome(t, t.TempDir())
	for _, original := range []string{"$(printf INJECTED)", "a'b \"c;`printf INJECTED`", "first\nsecond"} {
		cmd := GrokBotLaunchCommand(original, "direct", "https://api.example/v1", original)
		args := grokBotArgs(t, cmd)
		if grokBotArg(args, "--key") != original || grokBotArg(args, "--model") != original {
			t.Fatalf("shell interpreted argument: %#v", args)
		}
		if !strings.HasSuffix(args[0], "grok-switch-core.cjs") {
			t.Fatalf("downloaded full UI bundle: %s", args[0])
		}
	}
}

func TestGrokBotProviderProtocolAndHeaders(t *testing.T) {
	testenv.SetHome(t, t.TempDir())
	cases := []struct {
		p        provider.Provider
		protocol string
	}{
		{provider.Provider{ID: "chat", Name: "Chat", Chat: "https://anthropic.proxy.example/v1", Key: "synthetic", Models: []string{"model"}, Headers: map[string]string{"x-test": "a'b $(printf INJECTED)"}}, "openai-chat"},
		{provider.Provider{ID: "responses", Name: "Responses", Responses: "https://api.example/v1", Key: "synthetic", Models: []string{"model"}}, "openai-responses"},
		{provider.Provider{ID: "messages", Name: "Messages", Anthropic: "https://api.example/v1", Key: "synthetic", Models: []string{"model"}}, "anthropic-messages"},
	}
	for _, c := range cases {
		if err := provider.Save(c.p); err != nil {
			t.Fatal(err)
		}
		args := grokBotArgs(t, GrokBotLaunchCommand("magpie/"+c.p.ID+"/model", "direct", "", ""))
		if got := grokBotArg(args, "--protocol"); got != c.protocol {
			t.Fatalf("%s: protocol %q", c.p.ID, got)
		}
		if grokBotArg(args, "--model") != "model" {
			t.Fatal("direct request retained gateway provider prefix")
		}
		if c.p.Headers != nil && !slices.Contains(args, "x-test: a'b $(printf INJECTED)") {
			t.Fatal("lost provider header")
		}
	}
	if got := GrokBotLaunchCommand("unknown", "direct", "", ""); got != "" {
		t.Fatal("unresolved model fell back to localhost")
	}
}
