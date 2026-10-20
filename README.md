<div align="center">

<a href="https://useEcho.ai"><img src="site/public/img/icon-256.png" width="112" alt="Echo"></a>

# Echo

> **Echo is magpie, rebuilt on MyGo.** This project is a MyGo refactor of
> [magpie](https://usemagpie.ai): the same gateway, the same providers and
> the same agents, with a native-UI desktop shell, and some of its features
> simplified or made better.

### Every agent's model. One place.

**Claude Code on Kimi. Codex on DeepSeek. Gemini CLI on GLM. OpenCode on your ChatGPT plan.**<br>
Switch any of them from the menu bar. One local gateway serves them all,<br>
and when a quota runs out, it quietly moves on to the next account.

<!-- [![Release](https://img.shields.io/github/v/release/yetone/Echo-releases?label=release&color=111111)](https://github.com/yetone/Echo-releases/releases/latest) [![Stars](https://img.shields.io/github/stars/yetone/Echo?style=flat&color=111111)](https://github.com/yetone/Echo/stargazers) [![Discord](https://img.shields.io/badge/Discord-join-5865F2?logo=discord&logoColor=white)](https://discord.gg/vGSnD3ZKQF) [![License](https://img.shields.io/badge/license-MIT-111111)](LICENSE)<br>
![macOS](https://img.shields.io/badge/macOS-000000?logo=apple&logoColor=white) ![Windows](https://img.shields.io/badge/Windows-0078D4?logo=windows&logoColor=white) ![Linux](https://img.shields.io/badge/Linux-FCC624?logo=linux&logoColor=black) ![Docker](https://img.shields.io/badge/Docker-2496ED?logo=docker&logoColor=white) ![Termux](https://img.shields.io/badge/Termux-000000?logo=android&logoColor=white) -->
<!-- 
<sub>Sponsors</sub><br>
<a href="https://useEcho.ai/go/apinebula/readme"><picture><source media="(prefers-color-scheme: dark)" srcset="site/public/img/sponsor-apinebula-dark.svg"><img src="site/public/img/sponsor-apinebula.svg" height="36" alt="APINEBULA"></picture></a>
<a href="https://useEcho.ai/go/zerotoken/readme"><picture><source media="(prefers-color-scheme: dark)" srcset="site/public/img/sponsor-zerotoken-dark.svg"><img src="site/public/img/sponsor-zerotoken.svg" height="36" alt="零词元"></picture></a> -->

<!-- **[Download](https://useEcho.ai)** &nbsp;·&nbsp; **[Docs](https://useEcho.ai/docs/start)** &nbsp;·&nbsp; **[Full reference (English)](docs/reference.md)** &nbsp;·&nbsp; **[Plugins](https://github.com/Echo-community/plugins)** &nbsp;·&nbsp; **[Discord](https://discord.gg/vGSnD3ZKQF)** &nbsp;·&nbsp; [English](README.md) · **简体中文** -->

<br>

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/agents-dark.png">
  <img src="site/public/img/agents-light.png" width="900" alt="Echo's Agents page: Claude Code on Kimi K3, Codex on DeepSeek V4 Pro, Gemini CLI on GLM-5.3, each picked from one list">
</picture>

</div>

<br>

## Download and install

**macOS 13+**: download the arm64 DMG for Apple silicon or the x86_64 DMG for Intel Macs from [GitHub Releases](https://github.com/aohun/Echo/releases/latest).

**Windows x64**: download the Windows .exe installer from [GitHub Releases](https://github.com/aohun/Echo/releases/latest).

**Homebrew** is the recommended installation and update method on macOS:

```sh
brew tap aohun/echo https://github.com/aohun/Echo.git
brew install --cask aohun/echo/echo
brew upgrade --cask aohun/echo/echo
```

### If macOS says it cannot open Echo.app

If macOS says that Apple cannot verify whether `Echo.app` contains malware, the manually downloaded package may not yet be notarized by Apple. First confirm that the app came from the trusted [GitHub Releases](https://github.com/aohun/Echo/releases/latest), then use one of these methods:

1. In **Applications**, Control-click `Echo.app`, choose **Open**, and choose **Open** again in the confirmation dialog.
2. If macOS still blocks it, open **System Settings > Privacy & Security**, click **Open Anyway** next to the security warning, and launch the app again.
3. You can also remove the quarantine attribute in Terminal:

   ```sh
   sudo xattr -dr com.apple.quarantine /Applications/Echo.app
   ```

Only use these steps for an app whose source you trust. Homebrew installations usually do not require manual quarantine removal.

## Is Echo for you?

Each agent keeps its model in its own file, in its own format, with its own keys and base URLs. A subscription you pay for works in one agent and nowhere else. Echo puts all of that behind one app and one local gateway:

- **Pick any model for any agent** from one list. Echo edits the one key that matters in the agent's own config.
- **Use one provider everywhere**: an API key, a local model, or the Claude, ChatGPT, Copilot, Gemini or Grok plan you're signed in to.
- **Keep going when a quota runs out**: a routing group moves to the next account or model, and the agent never sees the error.

If you use more than one agent, more than one provider, or more than one account, it's for you. If you use one agent on its vendor's own plan and that's enough, you don't need Echo.
<!-- 
Runs on macOS, Windows and Linux (menu bar app, window, TUI, web UI and CLI), in Docker, and on Termux. -->

<!-- ## Quick start

**1 · Install.** Download the app from **[useEcho.ai](https://useEcho.ai)**, or run:

```sh
curl -fsSL https://useEcho.ai/install.sh | sh
```

<sub>Mac builds are signed and notarised, and every build updates itself. Behind a firewall, use `--proxy` or `--mirror`. Or `go install github.com/yetone/Echo@latest`, or the [Docker image](https://useEcho.ai/docs/docker).</sub>

**2 · Add a provider.** Open Echo and go to **Providers → Add provider**. Pick a preset and paste a key, or sign in with a subscription.

**3 · Pick a model** for each agent on the **Agents** page. Start a new session and it's on the new model.

Or do it all from the terminal:

```sh
Echo provider add deepseek sk-…              # a preset needs only the key
Echo claude deepseek/deepseek-v4-pro          # Claude Code on DeepSeek
Echo codex moonshot/kimi-k2.5                 # Codex on Kimi
Echo group add "Opus anywhere" models=claude/claude-opus-5-5,copilot/claude-opus-5.5 routing=smart
Echo claude group/opus-anywhere               # fails over between subscriptions
Echo save work && Echo use work             # profiles
Echo quota                                    # what's left on every plan
Echo tui                                      # the whole thing, in a terminal
```

Echo finds the subscriptions your agents are signed in to by reading their sign-ins (Claude Code's and Codex's files, the Keychain). To keep it from reading them, turn off Settings → Privacy → *Subscription accounts*, or run `Echo accounts discovery off` (`on` turns it back on). `Echo_NO_ACCOUNTS=1` turns it off for one run without changing the saved setting. API-key providers work either way. -->

## Supported agents

<table>
<tr><td>

Claude Code · Claude Desktop · Claude Science · Codex · Gemini CLI · Antigravity CLI · OpenCode · OpenChamber · MiMo Code · DevEco Code · Pi · oh-my-pi · StepCode · MothX · Aside · OmO · Prime Agent · Goose · Cursor CLI · Cursor Private Inference · Zed · VS Code Chat · VS Code Insiders · VSCodium Chat · JetBrains Air · Copilot (JetBrains) · Copilot CLI · Crush · DeepSeek Harness · Reasonix Studio · Command Code · fx · Devin · Hermes Agent · Mister Morph · Kimi Code · jcode · Qwen Code · Muse Code · Empryo · MiniMax Code · Droid · Cline · Qoder · Qoder CN · Grok Build · Grok App · ZCode · WorkBuddy · WorkBuddy AI · CodeBuddy Code · Pencil · OpenPencil · Raycast · T3 Code · OpenHanako · AtomCode · Snow CLI · Alma · Cindy · Lody · Ante · AstrBot

</td></tr>
</table>

Echo shows only the agents installed on your machine; setup notes for some of them are in the [reference](docs/reference.md#notes-on-some-agents). Anything else that takes a base URL can use the gateway too:

```sh
export OPENAI_BASE_URL=http://127.0.0.1:3426/v1     OPENAI_API_KEY=Echo
export ANTHROPIC_BASE_URL=http://127.0.0.1:3426     ANTHROPIC_API_KEY=Echo
export GOOGLE_GEMINI_BASE_URL=http://127.0.0.1:3426 GEMINI_API_KEY=Echo
```

## How it fits together

The gateway at `127.0.0.1:3426` speaks OpenAI Chat, OpenAI Responses, Anthropic Messages and Gemini, and translates between them, streaming, tool calls and reasoning included.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/architecture-dark.svg">
  <img src="site/public/img/architecture.svg" alt="How Echo fits together: your agents send requests to the Echo gateway at 127.0.0.1:3426, which routes them through routing groups and plugins to your API keys, subscriptions and local models.">
</picture>


## What it does

### Any model, any agent

Click a value and a searchable list opens: every model of every provider you added, as `provider/model`. The agent's config is rewritten atomically, and its comments, order and formatting stay as they were. **Profiles** save every agent's setup under one name ("Budget", "Focus") and switch them all back in one move. Each agent can also have its own short list of models; a profile keeps that list too.

<table>
<tr>
<td width="50%">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/panel-dark.png">
  <img src="site/public/img/panel-light.png" alt="The menu bar panel">
</picture>
</td>
<td width="50%">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/picker-dark.png">
  <img src="site/public/img/picker-light.png" alt="The model picker">
</picture>
</td>
</tr>
</table>

### Providers

Pick a preset, paste a key. Model lists come from the vendor, with names and reasoning levels filled in from [models.dev](https://models.dev), so a model released this morning is in your picker on the next refresh.

> **Presets include** Anthropic · OpenAI · Google Gemini · Amazon Bedrock (runtime and Mantle) · DeepSeek · Kimi · Zhipu GLM · MiniMax · StepFun · Qwen · Baidu Qianfan · Tencent Cloud · Huawei Cloud MaaS · Volcengine Ark · Mistral · Groq · Nous Portal · xAI · OpenRouter · Together · Fireworks · SiliconFlow · NVIDIA NIM · ModelScope · Ollama · LM Studio … and any OpenAI- or Anthropic-compatible URL.

**Import** brings in the providers from CC Switch, Claude Code, Codex and Alma. **[Add to Echo](https://useEcho.ai/docs/import)** links let a provider's website hand its config to Echo in one click. Each key and each account can go through a proxy of its own.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/add-dark.png">
  <img src="site/public/img/add-light.png" width="900" alt="Add provider: subscriptions to sign in to, vendors, relays and local servers">
</picture>

### Subscriptions

An agent you're signed in to is a subscription with models behind it, so Echo offers it as a provider to every other agent. Add several accounts per subscription and Echo fails over between them.

- **Claude**: drives the real local `claude` binary and bridges your agent's tools over MCP
- **Codex / ChatGPT**: your ChatGPT plan's models, in every other agent
- **GitHub Copilot**, **Gemini (Code Assist)**, **Antigravity**, **Grok (SuperGrok)** and more
- **Any OpenCode auth plugin or pi package** from npm

**Daily warm-up** starts each Claude and ChatGPT account's 5-hour window at the times you choose, or the moment the last one resets.

### Routing groups

A **routing group** is several models an agent picks as one, such as `group/daily-coding`. The gateway spreads requests over every member's keys and accounts:

| Mode | What it does |
| :-- | :-- |
| **`smart`** | Of the subscriptions with quota left, use the one that renews soonest, so less is wasted at the reset |
| **`order`** | Use the first model until it can't answer, then the next |
| **`rotate`** | Move to the next member on each turn |
| **`usage`** | Use the least-used member first |
| **`pace`** | Use the account with the most of its week left per hour until it resets |

A conversation **stays with the account that answered it** while the vendor's prompt cache is worth keeping. Groups can contain groups. **[Intent routing](https://useEcho.ai/docs/intent)** has a small model of your choice read each new turn, so tests go to the strong model and quick questions to the cheap one. The Routing view shows every decision live.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/routing-dark.png">
  <img width="760" src="site/public/img/routing-light.png" alt="The Routing view: four agents through Echo to seven providers, live">
</picture>

<table>
<tr>
<td width="50%">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/intent-trace-dark.png">
  <img src="site/public/img/intent-trace-light.png" alt="An intent-routed turn, explained step by step">
</picture>
</td>
<td width="50%">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/nested-routing-dark.png">
  <img src="site/public/img/nested-routing-light.png" alt="A routing group inside a routing group">
</picture>
</td>
</tr>
</table>

<!-- ### Plugins

```sh
Echo plugin add opencode-gemini-auth   # an OpenCode plugin from npm
Echo plugin add pi-antigravity         # a pi package, the same way
Echo plugin login google-plugin        # its own sign-in flow, in Echo
```

Plugins run on [Bun](https://bun.sh), downloaded the first time one needs it. A plugin signs in, lists its models and makes each request; agents use those models like any other provider's. Browse **[Echo-community/plugins](https://github.com/Echo-community/plugins)**, or [write your own](https://useEcho.ai/docs/plugins).

A plugin can also be **gateway middleware**: JavaScript that sees what every agent sends and gets back. `onRequest` can rewrite a request or turn it away, `onEvent` sees each streamed event, `onResponse` the whole reply. A hook that throws or runs too long leaves the request as it was.

```js
// alias.middleware.js — Echo plugin add ./alias.middleware.js
export function onRequest(body, ctx) {
  if (body.model === "fast") return { ...body, model: "deepseek/deepseek-chat" };
}
```

<details>
<summary><b>Ready-made middleware</b>, most of it what <a href="https://github.com/QuantumNous/new-api">New API</a> does for its channels, with the same JSON</summary>

<br>

| Package | What it does |
| :-- | :-- |
| `param-override` | New API's `param_override`: set, delete, move or rewrite request fields, under conditions, or turn a request away |
| `model-map` | New API's `model_mapping`: send a model under another name; replies keep the name asked for |
| `system-prompt` | Your system prompt on every request, or some agents' or models' |
| `word-guard` | New API's sensitive-word filter: turn away or mask words in what users send, and in replies |
| `think-tags` | Take `<think>…</think>` out of replies, or put `reasoning_content` into them |

```sh
Echo plugin add @Echo-community/middleware-model-map
Echo plugin options model-map '{"mapping": {"fast": "deepseek/deepseek-chat"}}'
```

Find them under Plugins › Discover › Gateway middleware. See [Gateway middleware](https://useEcho.ai/docs/plugins#middleware).

</details> -->

<!-- ### Usage and cost

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="site/public/img/usage-dark.png">
  <img src="site/public/img/usage-light.png" width="900" alt="The Usage page: balances, tokens, cache hit rate and cost">
</picture>

| | |
| :-- | :-- |
| **Tokens and cost** | Tokens, cache reads and writes, reasoning and calls, with cost at list price in dollars or yuan; set your own price for any model |
| **Quota** | Balances and quota windows for every key, plan and account (`Echo quota`), and a reminder before a window renews with much of it unused |
| **Sessions** | Every agent conversation with its cost and title, reopened in your terminal with one click |
| **Context** | How full each request's context window is, how much came from the cache, and what fills it |
| **Bills** | By account and by upstream key, and OTLP export to your own observability stack | -->

<!-- ### Sharing

| | |
| :-- | :-- |
| **Local network** | Turn on *Share on local network* and give each client a named gateway key with its own daily, weekly or monthly token and cost limit |
| **[Remote Echo](https://useEcho.ai/docs/remote)** | A laptop uses the providers, accounts and groups of the Echo on your desktop, and still wires its own agents |
| **[Docker](https://useEcho.ai/docs/docker)** | Run `ghcr.io/yetone/Echo` on a server or a NAS and manage it from the web UI |
| **Sync** | Back up to a file, or sync machines over WebDAV (Nutstore, Nextcloud…), S3, or an encrypted GitHub repository backup |

### Also

- **Library**: write instructions, MCP servers, skills and subagents once; Echo writes them into each agent's files in its format, and removes only what it wrote.
- **Web search for any model**: a model that can't search gets Echo's search: a provider that can, or search APIs (Tavily, Brave, Exa, Firecrawl, You.com, SearXNG), asked in the order you drag them into, each one able to be switched off without losing its key. An API can hold several keys: searches take turns over them (or use them in order with `pick=order`), a key refused lately is asked last, and Tavily keys show what each has used of its limit, in Settings and in `Echo search`. Any agent can mount the same search as an MCP server, with a `web_search` tool and a `web_fetch` tool that reads a page by its URL.
- **Images and video**: an image tool for any agent over MCP, with the model you choose, and videos with a Grok subscription.
- **Agents kept current**: the Agents page shows each CLI's version, updates it the way it was installed, and gives the commands for the ones you don't have.
- **WSL**: on Windows, agents inside WSL distros are wired too, and their sessions counted.
- **Keep awake**, light and dark, and your theme's look on [Omarchy](https://omarchy.org).

## Documentation

| | |
| :-- | :-- |
| **[Get started](https://useEcho.ai/docs/start)** | The guided tour |
| **[Full reference](docs/reference.md)** (English) | Every agent, provider option, gateway endpoint, CLI command and file |
| **[Plugins](https://useEcho.ai/docs/plugins)** | Use plugins and write your own |
| **[Intent routing](https://useEcho.ai/docs/intent)** | Route each turn by what it asks for |
| **[Import links](https://useEcho.ai/docs/import)** | "Add to Echo" buttons for provider websites |
| **[Remote Echo](https://useEcho.ai/docs/remote)** | Use one Echo's providers and groups from your other computers |
| **[Docker](https://useEcho.ai/docs/docker)** | Run Echo on a server or a NAS | -->
<!-- 
## Privacy

Your prompts, replies, keys and accounts go only to the providers you use. Once a day a released Echo tells us it is in use: a random id, its version and system, and which agents, providers and models it is used with, by Echo's own ids (a provider you added yourself is only `custom`), and for each partner listed first in the add sheet, how many times a day it was shown, opened and added (counts only). No names, URLs, accounts, keys, prompts or usage. Turn part or all of it off in **Settings → Privacy**, or with `DO_NOT_TRACK=1`. A partner's website and key links go through `useEcho.ai/go/…`, which counts the click, and useEcho.ai counts fetches of the partner list; the server sees these whatever the settings, and keeps no id or address. Icons named by a plugin, an import link or the library are fetched through `useEcho.ai/api/icon`, so their hosts see Cloudflare's address, not yours; the server only passes the picture on, and counts and keeps nothing. The [Privacy page](https://useEcho.ai/docs/privacy) lists everything Echo sends by itself, with an example and every switch; the [reference](docs/reference.md#counting-users) has the rules. -->

<!-- 
## Community

Questions, ideas, or a model that won't show up? Tell us on **[Discord](https://discord.gg/vGSnD3ZKQF)**. That is where feedback goes.

**This repository doesn't take pull requests.** Only maintainers can open them. If you'd like a fix or a feature, describe it on Discord and we'll build it.

If Echo saved you from editing one more config file, **a ⭐ helps others find it.**

<a href="https://star-history.com/#yetone/Echo&Date">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="https://api.star-history.com/svg?repos=yetone/Echo&type=Date&theme=dark">
    <img src="https://api.star-history.com/svg?repos=yetone/Echo&type=Date" width="600" alt="Star history">
  </picture>
</a>

## License

[MIT](LICENSE) -->
