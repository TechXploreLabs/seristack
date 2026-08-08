# Demo 1 — Shell workflows via CLI, HTTP, and AI agent (Gemini + MCP)

This demo shows the same seristack stacks being triggered three ways:
- Directly from the CLI
- Via HTTP API
- By a Gemini AI agent through the MCP server

## Prerequisites

Clone the repo and ensure seristack is installed.

## 1. Trigger stacks from the CLI

```bash
seristack trigger --config config.yaml --vars user_name=seristack --vars env_name=dev
```

Trigger a specific stack only:

```bash
seristack trigger --config config.yaml --vars user_name=seristack -s stack1
```

## 2. Run the HTTP API server

```bash
seristack run --config config.yaml --addr 127.0.0.1 --port 8080
```

Trigger a stack via HTTP:

```bash
curl -X POST http://127.0.0.1:8080/stack1 \
  -H "Content-Type: application/json" \
  -d '{"user_name": "seristack"}'
```

## 3. Run the MCP server

```bash
seristack mcp -t streamableHTTP --addr 127.0.0.1 --port 8080
```

## 4. Run the Gemini AI agent (Streamlit)

The Streamlit app connects to the MCP server and uses Gemini to call stacks
as tools based on natural language prompts.

Install dependencies:

```bash
pip3 install -r requirements.txt
```

Get a Gemini API key: https://ai.google.dev/gemini-api/docs, update the .env file with the api key

Set the key and start the app:

```bash
streamlit run main.py
```
