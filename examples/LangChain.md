# tinyMem + LangChain Integration

You can easily use tinyMem with LangChain to add persistent, project-scoped memory to your chains and agents.

## Setup

tinyMem exposes an OpenAI-compatible API, so you can use the standard `ChatOpenAI` class.

### 1. Basic Chain

```python
from langchain_openai import ChatOpenAI
from langchain.prompts import ChatPromptTemplate
import os

# Point to tinyMem proxy
# tinyMem handles the actual backend connection (OpenAI, Ollama, etc.)
llm = ChatOpenAI(
    base_url="http://localhost:8080/v1",
    api_key="dummy", # tinyMem ignores this, but LangChain requires it
    model="gpt-4o",  # Match the model name configured in .tinyMem/config.toml
    temperature=0
)

# Create a simple chain
prompt = ChatPromptTemplate.from_template("Based on our project context, {query}")
chain = prompt | llm

# Invoke - tinyMem will inject memories relevant to "database schema"
response = chain.invoke({"query": "What is the database schema?"})
print(response.content)
```

### 2. LangGraph Agent

LangGraph agents can also benefit from tinyMem's context injection.

```python
from langgraph.prebuilt import create_react_agent
from langchain_core.tools import tool

# Define your tools
@tool
def get_weather(city: str):
    """Get the weather for a city."""
    return "Sunny"

tools = [get_weather]

# Initialize LLM via tinyMem
llm = ChatOpenAI(base_url="http://localhost:8080/v1", api_key="dummy")

# Create agent
agent_executor = create_react_agent(llm, tools)

# Run
response = agent_executor.invoke({"messages": [("user", "What should I wear based on the weather?")]})
```

## Advanced: Accessing Recall Headers

tinyMem returns headers indicating what was recalled. To access these in LangChain, you may need to inspect the raw output or use a custom callback handler, as standard `ChatOpenAI` abstracts headers away.
