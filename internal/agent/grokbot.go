package agent

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/provider"
)

const GrokBotID = "grok-bot"
const GrokSwitchCoreScript = "grok-switch-core.cjs"

// A cloud agent cannot reach this computer through its own loopback address.
func grokBotCloudURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return errors.New("Set a cloud-accessible HTTP or HTTPS gateway URL")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	ip := net.ParseIP(host)
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "localhost.localdomain" || host == "0" || strings.HasPrefix(host, "127.") || (ip != nil && (ip.IsLoopback() || ip.IsUnspecified())) {
		return errors.New("The cloud computer cannot reach a gateway at localhost")
	}
	return nil
}

func grokBotShell(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }

// GrokBotLaunchCommand returns an empty command when cloud configuration is incomplete.
func GrokBotLaunchCommand(model, mode, targetURL, apiKey string) string {
	cmd, _ := grokBotCommand(model, mode, targetURL, apiKey)
	return cmd
}

func grokBotCommand(model, mode, targetURL, apiKey string) (string, error) {
	if model == "" {
		return "", errors.New("Choose a model for Grok Bot")
	}
	if mode == "" {
		mode = "direct"
	}
	ref := strings.TrimPrefix(model, magpieID+"/")
	name, modelName, protocol := "magpie", ref, "openai-chat"
	var headers map[string]string
	switch mode {
	case "gateway":
		if err := grokBotCloudURL(targetURL); err != nil {
			return "", err
		}
		if apiKey == "" {
			apiKey = access.LANSecret()
		}
		if apiKey == "" {
			return "", errors.New("Enable an existing gateway sharing key before copying the Grok Bot prompt")
		}
	case "direct":
		if strings.HasPrefix(ref, provider.GroupPrefix) {
			return "", errors.New("Model groups require gateway mode")
		}
		if p, actual, ok := provider.Resolve(ref); ok {
			if p.Account != nil || p.PluginProvider() != "" {
				return "", errors.New("Subscription and plugin providers require gateway mode")
			}
			name, modelName, headers = p.ID, actual, p.Headers
			pr := p.Native(actual)
			if p.BaseAPI != "" {
				pr = provider.Protocol(p.BaseAPI)
			}
			switch pr {
			case provider.Chat:
				protocol = "openai-chat"
			case provider.Responses:
				protocol = "openai-responses"
			case provider.Anthropic:
				protocol = "anthropic-messages"
			default:
				return "", errors.New("This provider requires gateway mode")
			}
			if targetURL == "" {
				targetURL = p.Base(pr)
			}
			if apiKey == "" {
				apiKey = p.Key
			}
		}
		if err := grokBotCloudURL(targetURL); err != nil {
			return "", err
		}
		if apiKey == "" {
			return "", errors.New("Set an API key for the selected provider")
		}
	default:
		return "", errors.New("Choose direct or gateway mode")
	}
	args := []string{"use", name, "--url", targetURL, "--model", modelName, "--protocol", protocol, "--key", apiKey}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--header", k+": "+headers[k])
	}
	for i := range args {
		args[i] = grokBotShell(args[i])
	}
	return fmt.Sprintf("mkdir -p /workspace/grok-switch && curl -fsSL 'https://raw.githubusercontent.com/enderzcx/grok-bot-switch/main/dist/%s' -o /workspace/grok-switch/%s && node /workspace/grok-switch/%s %s", GrokSwitchCoreScript, GrokSwitchCoreScript, GrokSwitchCoreScript, strings.Join(args, " ")), nil
}

func GrokBotPrompt(model, cmd string) string {
	return "请在你的云端电脑执行下面这条命令，然后把完整输出原样发给我：\n\n" + cmd
}

func grokBot(home string) *Agent { return grokBotIn(here(home)) }

func grokBotIn(at place) *Agent {
	dir := at.getenv("GROK_BOT_HOME")
	if dir == "" {
		dir = filepath.Join(at.home, ".grok-bot")
	}
	path := filepath.Join(dir, "config.json")
	getModel, getMode, getURL := jsonGet(path, "model"), jsonGet(path, "mode"), jsonGet(path, "gatewayURL")
	command := func() (string, error) {
		target := ""
		if getMode() == "gateway" {
			target = getURL()
		}
		return grokBotCommand(getModel(), getMode(), target, "")
	}
	return atomic(&Agent{
		ID: GrokBotID, Name: "Grok Bot", Icon: "xai",
		Aliases: []string{"grokbot", "grok-cloud", "grok-switch"},
		UA:      []string{"grok-switch", "grok-bot"}, Spelled: prefixed, Dir: dir, Path: path,
		detect: func() bool { return !Taken(dir) },
		Launch: func() string {
			cmd, err := command()
			if err != nil {
				return ""
			}
			return GrokBotPrompt(getModel(), cmd)
		},
		Notice: func() string {
			if _, err := command(); err != nil {
				return err.Error()
			}
			return "Send the installation prompt to Grok Bot; use /gs use to switch providers or /gs official to return to Grok"
		},
		Fields: []Field{
			{Key: "model", Label: "model", Get: getModel, Set: jsonSet(path, "model"), Options: func(cur map[string]string) []Option { return viaMagpie(GrokBotID, magpieID+"/") }},
			{Key: "mode", Label: "mode", Get: getMode, Set: func(v string) error {
				if v != "" && v != "direct" && v != "gateway" {
					return errors.New("Choose direct or gateway mode")
				}
				return jsonSet(path, "mode")(v)
			}, Options: func(cur map[string]string) []Option {
				return []Option{
					{Value: "direct", Label: "direct", Note: "Connect directly to an API-key provider"},
					{Value: "gateway", Label: "gateway", Note: "Connect through your cloud-accessible Magpie gateway"},
				}
			}},
			{Key: "gateway-url", Label: "Gateway URL", Get: getURL, Set: func(v string) error {
				if v != "" {
					if err := grokBotCloudURL(v); err != nil {
						return err
					}
				}
				return jsonSet(path, "gatewayURL")(v)
			}, Options: func(cur map[string]string) []Option { return nil }},
		},
	}, path)
}
