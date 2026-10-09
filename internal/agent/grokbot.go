package agent

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/provider"
)

// GrokBotID is the agent identifier for Grok Bot.
const GrokBotID = "grok-bot"

// GrokSwitchScript is the full grok-switch bundle with embedded UI.
const GrokSwitchScript = "grok-switch.cjs"

// GrokSwitchCoreScript is the headless core grok-switch bundle without UI.
const GrokSwitchCoreScript = "grok-switch-core.cjs"

// GrokBotLaunchCommand generates the one-click installation and configuration command
// for Grok Bot running in its cloud workspace.
func GrokBotLaunchCommand(model, mode, targetURL, apiKey string) string {
	if mode == "" {
		mode = "direct"
	}

	modelName := model
	ref, isMag := strings.CutPrefix(model, magpieID+"/")
	if isMag {
		modelName = ref
	}

	provID := ""
	protocol := "openai-chat"
	lookupRef := modelName
	if isMag {
		lookupRef = ref
	}

	if p, actualModel, ok := provider.Resolve(lookupRef); ok {
		provID = p.ID
		if actualModel != "" {
			modelName = actualModel
		}
		if mode == "direct" {
			if targetURL == "" {
				switch {
				case p.Chat != "":
					targetURL = p.Chat
					protocol = "openai-chat"
				case p.Responses != "":
					targetURL = p.Responses
					protocol = "openai-responses"
				case p.Anthropic != "":
					targetURL = p.Anthropic
					protocol = "anthropic-messages"
				}
			}
			if apiKey == "" {
				apiKey = p.Key
			}
		}
	}

	if mode == "gateway" {
		if targetURL == "" {
			targetURL = gateway.URL() + "/v1"
		}
		if apiKey == "" {
			apiKey = gateway.TokenFor(GrokBotID)
		}
		protocol = "openai-chat"
	}

	if targetURL == "" {
		targetURL = gateway.URL() + "/v1"
	}
	if apiKey == "" && mode == "gateway" {
		apiKey = gateway.TokenFor(GrokBotID)
	}
	if modelName == "" {
		modelName = "default"
	}

	if strings.Contains(targetURL, "anthropic") {
		protocol = "anthropic-messages"
	} else if strings.Contains(targetURL, "responses") {
		protocol = "openai-responses"
	}

	name := "magpie"
	if mode == "direct" && provID != "" {
		name = provID
	}

	flags := []string{"use", name}
	if targetURL != "" {
		flags = append(flags, "--url", targetURL)
	}
	if modelName != "" {
		flags = append(flags, "--model", modelName)
	}
	if protocol != "" && protocol != "openai-chat" {
		flags = append(flags, "--protocol", protocol)
	}
	if apiKey != "" {
		flags = append(flags, "--key", apiKey)
	}

	return fmt.Sprintf(
		"mkdir -p /workspace/grok-switch && curl -fsSL \"https://raw.githubusercontent.com/enderzcx/grok-bot-switch/main/dist/%s?v=$(date +%%s)\" -o /workspace/grok-switch/%s && node /workspace/grok-switch/%s %s",
		GrokSwitchScript, GrokSwitchScript, GrokSwitchScript, strings.Join(flags, " "),
	)
}

// GrokBotPrompt wraps the launch command into a prompt for the user to send to Grok Bot.
func GrokBotPrompt(model, cmd string) string {
	return "请在你的云端电脑执行下面这条命令，然后把完整输出原样发给我：\n\n`" + cmd + "`"
}

// grokBotPresets returns common model presets for Grok Bot.
func grokBotPresets() []Option {
	seen := map[string]bool{}
	var out []Option
	add := func(val, label, direct, icon, note string) {
		if val == "" || seen[val] {
			return
		}
		seen[val] = true
		out = append(out, Option{Value: val, Label: label, Icon: icon, Direct: direct, Note: note})
	}
	if ms, _, _ := catalog.Live("grok"); len(ms) > 0 {
		for _, m := range ms {
			add(m.ID, m.Name, "xAI", "xai", "xAI Grok")
		}
	}
	add("grok-4.6", "Grok 4.6", "xAI", "xai", "Official Grok model")
	add("grok-4-mini", "Grok 4 Mini", "xAI", "xai", "Official Grok model")
	add("gpt-5", "GPT-5", "OpenAI", "openai", "OpenAI ChatGPT/API")
	add("claude-opus-4-6", "Claude Opus 4.6", "Anthropic", "anthropic", "Anthropic Claude")
	add("deepseek-reasoner", "DeepSeek Reasoner", "DeepSeek", "deepseek", "DeepSeek API")
	return group("Grok Bot", out)
}

func grokBot(home string) *Agent { return grokBotIn(here(home)) }

func grokBotIn(at place) *Agent {
	dir := at.getenv("GROK_BOT_HOME")
	if dir == "" {
		dir = filepath.Join(at.home, ".grok-bot")
	}
	path := filepath.Join(dir, "config.json")
	getModel := jsonGet(path, "model")
	setModel := jsonSet(path, "model")
	getMode := jsonGet(path, "mode")
	setMode := jsonSet(path, "mode")

	return atomic(&Agent{
		ID:      GrokBotID,
		Name:    "Grok Bot",
		Icon:    "xai",
		Aliases: []string{"grokbot", "grok-cloud", "grok-switch"},
		UA:      []string{"grok-switch", "grok-bot"},
		Spelled: prefixed,
		Dir:     dir,
		Path:    path,
		Launch: func() string {
			m := getModel()
			mode := getMode()
			if mode == "" {
				mode = "direct"
			}
			var targetURL string
			if mode == "gateway" {
				targetURL = at.v1()
			}
			cmd := GrokBotLaunchCommand(m, mode, targetURL, "")
			return GrokBotPrompt(m, cmd)
		},
		Notice: func() string {
			return "在 Grok Bot 云端会话中执行提示词即可生效；支持通过 /gs use 切换供应商，通过 /gs official 切回官方 Grok。"
		},
		Fields: []Field{
			{
				Key:   "model",
				Label: "model",
				Get:   getModel,
				Set:   setModel,
				Options: func(cur map[string]string) []Option {
					return append(viaMagpie(GrokBotID, magpieID+"/"), grokBotPresets()...)
				},
			},
			{
				Key:   "mode",
				Label: "mode",
				Get:   getMode,
				Set: func(v string) error {
					if v == "" {
						return edit.DelJSON(path, "mode")
					}
					return setMode(v)
				},
				Options: func(cur map[string]string) []Option {
					return []Option{
						{Value: "direct", Label: "direct", Note: "Direct connection to upstream provider (default)"},
						{Value: "gateway", Label: "gateway", Note: "Relay through local magpie gateway"},
					}
				},
			},
		},
	}, path)
}
