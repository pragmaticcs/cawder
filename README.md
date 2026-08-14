# Cawder

Cawder is a minimal coding agent harness designed primarily for local models, while also supporting cloud providers exposing an OpenAI-compatible API.

<img src="demo/tldr.gif" alt="Cawder TUI Demo">
    
## Features

- **Agent Capabilities**: provides the model with unsandboxed tools for reading, writing, editing files and command execution.
- **Plug and Play Design**: supports most LLM inference engines that expose an OpenAI-compatible API, allowing for use with both local and cloud models. 
- **Simple Configuration**: uses one `conf.toml` file for specifying models along with servers, context window and API keys.
- **Familiar UI**: features a minimal TUI optimized for speed providing the features that you actually need: markdown rendering, model selection menu, resuming sessions.  
- **Built for Speed**: implemented in Go to leverage its concurrency capabilities and efficient I/O operations.
- **Minimal Dependencies**: utilizes the popular [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI framework and the official [OpenAI Go API Library](https://github.com/openai/openai-go). 

## Security

> **WARNING:** Don't use this in an environment containing anything you care about. Seriously. Cawder gives the LLM unsandboxed access to your filesystem and command execution. It can potentially read, modify, delete, or execute anything accessible to the Cawder process.

Cawder is an experimental, work-in-progress project intended for educational purposes and personal use. It has not been extensively tested. 

## Installation

**Requirements**: Go 1.24+, Git, a terminal emulator, and an OpenAI-compatible inference server.

The codebase is tiny, so building from source is the simplest way:

1) Clone the repo:
```sh
git clone https://github.com/pragmaticcs/cawder 
```

2) Run the installation script contained in the cloned repo:

```sh
./install.sh
```
> Support for Windows coming soon.

3) Run cawder in your project, which will initialize a new `.cawder` directory with a blank config file (`conf.toml`):
```sh
cawder
```

Using the harness with a local model requires OpenAI-compatible server running on your machine or network. 

## Configuration

Add or remove model profiles by editing the `conf.toml` file in the `.cawder` directory:

```toml
selected = "qwen" # default model profile

[models]
    [models.qwen]
        name = "Qwen/Qwen3.6-35B-A3B" # model identifier
        url = "http://127.0.0.1:8888/v1" # API base URL
        key = "API_KEY" # API key (leave as empty "" when the inference server does not require authentication)
        context = 32000 # maximum context window Cawder assumes
    [models.mistral]
        name = "mistral-medium-3-5"
        url = "https://api.mistral.ai/v1"
        key = "API_KEY"
        context = 256000
```

> **NOTE**: This won't automatically download the model for you. The model has to be loaded in the inference engine of your choice before you can use it in the harness. Configuration is only needed so that the harness knows how to communicate correctly with the API.

Setting the system prompt is done by creating a `SYSTEM.md` file in the `.cawder` directory. The full contents of `SYSTEM.md` will be added to the minimal default system instructions. 

## Choosing a local LLM inference server 

Here are some LLM inference servers that expose an OpenAI-compatible API: 
- [Llama.cpp](https://github.com/ggml-org/llama.cpp)
- [Unsloth Studio](https://unsloth.ai/docs/new/studio)
- [LM Studio](https://lmstudio.ai/) 
- [Mistral.rs](https://github.com/ericlbuehler/mistral.rs)
- [vLLM](https://vllm.ai/)

## Limitations and Roadmap

The following features are not yet implemented but are on my roadmap:
- Skills support
- Web search tool
- MCP support
- Sub-agents

## FAQ

**Q: Why "Cawder"?**

A: The name cawder was inspired by the `Caw! Caw!` sound that crows make. Because large language models are just like crows: very smart, like collecting coins but you shouldn't trust them with managing your production codebase.

**Q: Is this harness better in any way than my favourite harness?**

A: Probably not. Unless your favourite harness has been compromised by some massive supply-chain attack as a result of a zero-day in one of its dependencies (just a joke btw, can happen to anybody). 

**Q: Then why create your own harness?**

A: Because most other harnesses still require customization in the form of writing or downloading plugins to get to an acceptable level of real productivity. So if I am forced to write extensions, I'd rather write them for my own harness. The goal of this project isn't to outperform established harnesses but to provide a small, understandable codebase with a minimal dependency footprint that I can audit and modify myself.

**Q: Was this vibe-coded?**

A: Ironically, no. I needed a codebase I understand and trust, and vibe-coding by definition requires you to not worry about the code. Though I did use AI to write some boilerplate and help with the parser.

## Acknowledgements

Inspired by these projects:
- [codehamr](https://codehamr.com/)
- [pi](https://pi.dev/)
- [minion](https://github.com/Sentdex/minion)

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
