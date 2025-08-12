# GromitAI 🤖⚡️

An AI-powered CLI tool that helps you find and execute the right terminal commands based on natural language queries. GromitAI uses OpenAI's GPT models to understand your intent and generate appropriate Linux commands, with the option to execute them directly.

## Features

- 🧠 **AI-Powered Command Generation**: Uses OpenAI's GPT models to understand natural language queries
- 🔍 **Smart Command Discovery**: Finds the right terminal commands for your specific needs
- ⚡️ **Interactive Execution**: Asks for confirmation before running commands
- 🎯 **Linux Environment Focus**: Optimized for Linux/Unix command line operations
- 🔧 **Configurable**: Customizable AI models and system prompts (in future)

## Installation

### Prerequisites

- Go 1.24.5 or higher
- OpenAI API key

### Building from Source

1. Clone the repository:
```bash
git clone https://github.com/amirhd23/gromitai.git
cd gromitai
```

2. Build the project:
```bash
go build -o gromit
```

3. Make it executable and move to your PATH:
```bash
chmod +x gromit
sudo mv gromit /usr/local/bin/
```

## Configuration

### OpenAI API Key

Set your OpenAI API key as an environment variable:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

## Usage

### Basic Usage

Simply describe what you want to do, and GromitAI will find the appropriate command:

```bash
gromit "list all files in the current directory"
```

### Interactive Mode

GromitAI will:
1. Generate the appropriate command
2. Show you what it's going to run
3. Ask for confirmation
4. Execute the command if you approve

### Examples

```bash
# System information
gromit "show me disk usage for all mounted filesystems"
```

## Command Line Options

```bash
gromit [options] "your query here"
```

### Available Flags

- `--agent`: AI agent to use (default: "openai")
- `--model`: AI model to use (default: "gpt-4o")
- `--systemPrompt`: Custom system prompt for the AI agent

### Examples with Options

```bash
# Use a different model
gromit --model gpt-4 "find all files modified today"

# Custom system prompt
gromit --systemPrompt "You are a security expert" "check for suspicious network connections"
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

⚠️ **Use with caution**: This tool executes real system commands. Always review the generated commands before confirming execution. The authors are not responsible for any damage caused by executed commands.
