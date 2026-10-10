Grok Bot support in Sunny's Magpie fork. Choose a configured provider, then copy the installation prompt to Grok Bot's cloud chat. Direct mode uses API-key providers; gateway mode requires a cloud-accessible URL and an enabled sharing key. Subscription accounts and model groups use gateway mode.

The cloud installer uses grok-bot-switch v0.8.6's headless core. Arguments are quoted literally, gateway models preserve their provider/group, and incomplete settings show an actionable notice instead of an unusable prompt. The fork updates from this repository, with SHA-256 verification.

## Install

Download the asset for your platform. macOS ZIPs contain Magpie.app; Windows EXEs and Linux desktop binaries include the GUI. Assets named magpie-cli-* are terminal only. macOS packages are ad-hoc signed and are not Apple-notarized; this fork does not have the upstream signing credentials.

Real cloud-account and ChatGPT OAuth acceptance remains unverified. Automated checks cover installation, upgrade, restoration, protocol routing, shell arguments, and GUI copying.

<!-- lang:zh -->
Grok Bot 已接入 Sunny 的 Magpie 二开版。选择已配置的供应商后，复制安装提示词发送到 Grok Bot 云端会话。直连模式用于 API 密钥供应商；网关模式需要云端可访问的地址，以及已启用的共享密钥。订阅账号和模型组使用网关模式。

云端安装使用 grok-bot-switch v0.8.6 的无面板核心包。命令参数保持原样，网关模型保留供应商或模型组，不完整的配置会显示提示。此二开版从本仓库更新，并验证 SHA-256。

## Install

按平台下载。macOS ZIP 内含 Magpie.app；Windows EXE 和 Linux 桌面文件包含界面。magpie-cli-* 是纯命令行版本。macOS 包使用临时签名，未经 Apple 公证；二开版没有原作者的签名凭据。

真实云端账号和 ChatGPT OAuth 验收尚未完成。自动检查覆盖安装、升级、恢复、协议选择、命令参数和界面复制。
