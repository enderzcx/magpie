package agent

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestGrokBot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("GROK_BOT_HOME", filepath.Join(home, ".grok-bot"))

	if err := provider.Save(provider.Provider{
		ID:   "deepseek",
		Name: "DeepSeek",
		Chat: "https://api.deepseek.com/v1",
		Key:  "ds-secret-key",
		Models: []string{
			"pro",
			"flash",
		},
	}); err != nil {
		t.Fatal(err)
	}

	a := grokBot(home)

	// 1. Agent 基础属性 (ID, Name, Aliases, Icon, UA)
	if a.ID != GrokBotID {
		t.Fatalf("expected ID %q, got %q", GrokBotID, a.ID)
	}
	if a.Name != "Grok Bot" {
		t.Fatalf("expected Name \"Grok Bot\", got %q", a.Name)
	}
	if a.Icon != "xai" {
		t.Fatalf("expected Icon \"xai\", got %q", a.Icon)
	}
	wantAliases := []string{"grokbot", "grok-cloud", "grok-switch"}
	for _, alias := range wantAliases {
		if !slices.Contains(a.Aliases, alias) {
			t.Errorf("expected alias %q in %v", alias, a.Aliases)
		}
	}
	wantUA := []string{"grok-switch", "grok-bot"}
	for _, ua := range wantUA {
		if !slices.Contains(a.UA, ua) {
			t.Errorf("expected UA %q in %v", ua, a.UA)
		}
	}

	// Find resolution
	for _, q := range []string{"grok-bot", "grokbot", "grok-cloud", "grok-switch"} {
		found, err := Find(q)
		if err != nil || found.ID != GrokBotID {
			t.Fatalf("Find(%q) failed: got %v, err %v", q, found, err)
		}
	}

	// 2. 模型获取与设置 (Field("model") 的 Get/Set)
	fModel := a.Field("model")
	if fModel == nil {
		t.Fatal("missing field 'model'")
	}
	if got := fModel.Get(); got != "" {
		t.Fatalf("expected empty model initially, got %q", got)
	}
	if err := fModel.Set("magpie/deepseek/pro"); err != nil {
		t.Fatalf("set model failed: %v", err)
	}
	if got := fModel.Get(); got != "magpie/deepseek/pro" {
		t.Fatalf("expected model \"magpie/deepseek/pro\", got %q", got)
	}

	// Options
	opts := fModel.Options(a.Values())
	if len(opts) == 0 {
		t.Fatal("expected model options, got empty")
	}
	hasDeepSeek := false
	hasGrok46 := false
	for _, o := range opts {
		if o.Value == "magpie/deepseek/pro" {
			hasDeepSeek = true
		}
		if o.Value == "grok-4.6" {
			hasGrok46 = true
		}
	}
	if !hasDeepSeek {
		t.Errorf("expected magpie/deepseek/pro in options")
	}
	if !hasGrok46 {
		t.Errorf("expected grok-4.6 preset in options")
	}

	// Clear model
	if err := fModel.Set(""); err != nil {
		t.Fatalf("clear model failed: %v", err)
	}
	if got := fModel.Get(); got != "" {
		t.Fatalf("expected empty model after clear, got %q", got)
	}

	// 3. 模式获取与设置 (Field("mode"))
	fMode := a.Field("mode")
	if fMode == nil {
		t.Fatal("missing field 'mode'")
	}
	if got := fMode.Get(); got != "" {
		t.Fatalf("expected empty mode initially, got %q", got)
	}

	modeOpts := fMode.Options(a.Values())
	modeVals := make([]string, len(modeOpts))
	for i, o := range modeOpts {
		modeVals[i] = o.Value
	}
	if !slices.Contains(modeVals, "direct") || !slices.Contains(modeVals, "gateway") {
		t.Fatalf("expected direct and gateway options, got %v", modeVals)
	}

	if err := fMode.Set("gateway"); err != nil {
		t.Fatalf("set mode gateway failed: %v", err)
	}
	if got := fMode.Get(); got != "gateway" {
		t.Fatalf("expected mode \"gateway\", got %q", got)
	}

	if err := fMode.Set("direct"); err != nil {
		t.Fatalf("set mode direct failed: %v", err)
	}
	if got := fMode.Get(); got != "direct" {
		t.Fatalf("expected mode \"direct\", got %q", got)
	}

	if err := fMode.Set(""); err != nil {
		t.Fatalf("clear mode failed: %v", err)
	}
	if got := fMode.Get(); got != "" {
		t.Fatalf("expected empty mode after clear, got %q", got)
	}

	// 4. Launch() 函数生成的提示词包含核心命令结构
	if err := fModel.Set("magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	prompt := a.Launch()
	if !strings.Contains(prompt, "mkdir -p /workspace/grok-switch") {
		t.Fatalf("expected prompt to contain \"mkdir -p /workspace/grok-switch\", got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "grok-switch") {
		t.Fatalf("expected prompt to contain \"grok-switch\", got:\n%s", prompt)
	}
	if !strings.Contains(prompt, "use") {
		t.Fatalf("expected prompt to contain \"use\", got:\n%s", prompt)
	}

	// Test Launch with mode gateway
	if err := fMode.Set("gateway"); err != nil {
		t.Fatal(err)
	}
	promptGw := a.Launch()
	if !strings.Contains(promptGw, "mkdir -p /workspace/grok-switch") ||
		!strings.Contains(promptGw, "grok-switch") ||
		!strings.Contains(promptGw, "use") {
		t.Fatalf("expected gateway prompt to contain core commands, got:\n%s", promptGw)
	}

	// Helper functions test
	cmd := GrokBotLaunchCommand("gpt-5", "direct", "https://api.openai.com/v1", "sk-123456")
	if !strings.Contains(cmd, "mkdir -p /workspace/grok-switch") ||
		!strings.Contains(cmd, "grok-switch") ||
		!strings.Contains(cmd, "use") ||
		!strings.Contains(cmd, "https://api.openai.com/v1") ||
		!strings.Contains(cmd, "sk-123456") {
		t.Fatalf("unexpected GrokBotLaunchCommand output: %s", cmd)
	}

	p := GrokBotPrompt("gpt-5", cmd)
	if !strings.Contains(p, cmd) || !strings.Contains(p, "请在你的云端电脑执行下面这条命令") {
		t.Fatalf("unexpected GrokBotPrompt output: %s", p)
	}

	// Notice advice
	notice := a.Notice()
	if !strings.Contains(notice, "/gs use") || !strings.Contains(notice, "/gs official") {
		t.Fatalf("expected notice to mention /gs use and /gs official, got: %q", notice)
	}
}
