package main

import (
	"fmt"
	"os"
	"strings"
)

// memory_addContract adds the MANDATORY TINYMEM CONTROL PROTOCOL to agent markdown files
func memory_addContract() {
	fmt.Println("This function will add the MANDATORY TINYMEM CONTROL PROTOCOL to the following files:")
	fmt.Println("- AGENTS.md")
	fmt.Println("- QWEN.md")
	fmt.Println("- GEMINI.md")
	fmt.Println("- CLAUDE.md")
	fmt.Println()
	fmt.Println("It will append the contract to the end of each file if it doesn't already exist.")
	fmt.Println("It will also update README.md to include setup instructions for MCP usage.")
	fmt.Println()

	var response string
	fmt.Print("Are you happy to proceed? (yes/no): ")
	fmt.Scanln(&response)

	if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
		fmt.Println("Operation cancelled.")
		return
	}

	files := []string{"AGENTS.md", "QWEN.md", "GEMINI.md", "CLAUDE.md"}
	contractContent := getContractContent()

	for _, filename := range files {
		if _, err := os.Stat(filename); err == nil {
			// File exists, append contract if not already present
			appendContractToFile(filename, contractContent)
		} else {
			// File doesn't exist, create it with contract
			createFileWithContract(filename, contractContent)
		}
	}

	// Update README.md
	updateReadme()

	fmt.Println("Operation completed successfully!")
}

func getContractContent() string {
	bt := "`" // backtick for markdown code spans
	return `

---

## MANDATORY TINYMEM CONTROL PROTOCOL

You are not a conversational assistant. You are an executor operating inside a constrained system.
This protocol is **not optional**. Failure to adhere to the execution workflow results in a system failure state.

### 1. SCOPE OF PROTOCOL

This protocol applies to **any request** involving this repository, including:
*   Code generation or refactoring
*   Documentation and architecture
*   Tooling, configuration, or environment
*   Bug fixes, testing, and debugging
*   "Where is X?" or "How does Y work?" inquiries

**Exception:** Only trivial small talk (e.g., "Hello") may bypass this protocol.

---

### 2. EXECUTION WORKFLOW (STRICT ORDER)

You must follow these steps sequentially. Do not skip steps.

#### Step 0: Identify Concepts
Extract **2–6 project-specific nouns** from the user request.
*   **Valid:** ` + bt + `auth middleware` + bt + `, ` + bt + `SQLite FTS` + bt + `, ` + bt + `proxy port` + bt + `, ` + bt + `CLI flags` + bt + `, ` + bt + `user schema` + bt + `
*   **Invalid:** ` + bt + `implementation` + bt + `, ` + bt + `logic` + bt + `, ` + bt + `details` + bt + `, ` + bt + `code` + bt + `, ` + bt + `help` + bt + `

These concepts define your search parameters.

#### Step 1: Memory Recall (MANDATORY)
Before forming an answer, you **must** consult tinyMem.
*   **Action:** Call ` + bt + `memory_query(query="...")` + bt + ` using the concepts from Step 0.
*   **Broad Context:** If a project-wide view is needed, call ` + bt + `memory_query(query="")` + bt + `.
*   **Fallback:** If results are empty, call ` + bt + `memory_recent()` + bt + ` to inspect the latest entries.

**CRITICAL:** If no memory tool is called, you are **not allowed** to answer.

#### Step 2: Evidence Gathering
Memory is the map; the repo is the terrain. You must verify claims against current reality.
*   **Check:** Code paths, file existence, runtime behavior, configuration values.
*   **Action:** Use file read tools or shell tools to gather evidence.
*   **Constraint:** Do not guess. Do not rely on training data if repo evidence is available.

#### Step 3: Synthesis
Construct your response only after Steps 0–2 are complete.
*   **Combine:** Merge tinyMem results with verified repo evidence.
*   **Attribution:** Explicitly state what came from memory vs. what came from current files.
*   **Conflict Resolution:** Explicitly note if Memory says X but Code says Y.
*   **Empty State:** If memory was empty, explicitly state: *"No relevant memory found. Proceeding with repository evidence."*

#### Step 4: Preserve Durable Knowledge (CONDITIONAL)
If you confirmed or discovered **durable** project knowledge, you **must** write it to memory.
*   **What to write:** Facts, decisions, constraints, non-obvious conclusions, architectural patterns.
*   **How to write:** Call ` + bt + `memory_write` + bt + ` with a clear summary and optional details.
*   **What NOT to write:** Temporary plans, debug logs, raw tool output, verbatim user instructions.

#### Step 5: Memory Health (CONDITIONAL)
If the user asks about memory state, connectivity, or scope:
*   **Action:** Call ` + bt + `memory_health` + bt + ` before replying.
*   **Constraint:** Never guess the status of the memory database.

---

### 3. COMPLIANCE & ENFORCEMENT

#### Tool Honesty Rule
*   Never fabricate tool outputs.
*   Never imply a tool was called when it was not.
*   All tool calls must occur **before** the final response text is generated.

#### The Enforcement Invariant
For any project-related request:

> **A valid response must be preceded by at least one successful tinyMem memory call (` + bt + `memory_query` + bt + ` or ` + bt + `memory_recent` + bt + `) in the same generation run.**

If this invariant is violated, the response is structurally invalid.

---

### 4. MENTAL MODEL

1.  **TinyMem is the source of continuity.** It bridges the gap between sessions.
2.  **You are the interface.** Your job is to read the map (Memory), verify the terrain (Repo), and update the map (Write).
3.  **Silence is failure.** Falling back to generic training data without checking memory is a protocol violation.
`
}

func appendContractToFile(filename, contractContent string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading %s: %v\n", filename, err)
		return
	}

	fileContent := string(content)
	if strings.Contains(fileContent, "MANDATORY TINYMEM CONTROL PROTOCOL") {
		fmt.Printf("Contract already exists in %s, skipping.\n", filename)
		return
	}

	newContent := fileContent + contractContent
	err = os.WriteFile(filename, []byte(newContent), 0644)
	if err != nil {
		fmt.Printf("Error writing to %s: %v\n", filename, err)
		return
	}

	fmt.Printf("Contract appended to %s\n", filename)
}

func createFileWithContract(filename, contractContent string) {
	content := "# Agent Contract for tinyMem\n\n" + contractContent
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error creating %s: %v\n", filename, err)
		return
	}

	fmt.Printf("Created %s with contract\n", filename)
}

func updateReadme() {
	content, err := os.ReadFile("README.md")
	if err != nil {
		fmt.Printf("Error reading README.md: %v\n", err)
		return
	}

	readmeContent := string(content)
	if strings.Contains(readmeContent, "MANDATORY TINYMEM CONTROL PROTOCOL") {
		fmt.Println("Contract section already exists in README.md, skipping.")
		return
	}

	// Find a good place to insert the MCP setup instructions
	insertionPoint := strings.Index(readmeContent, "## IDE Integration")
	if insertionPoint == -1 {
		// If we can't find the IDE Integration section, append to end
		readmeContent += "\n\n## Setting Up Agents for MCP Usage\n\n"
		readmeContent += "When using tinyMem as an MCP server for AI agents, ensure that your agents follow the MANDATORY TINYMEM CONTROL PROTOCOL.\n\n"
		readmeContent += "Include the contract content from [AGENT_CONTRACT.md](AGENT_CONTRACT.md) in your agent's system prompt to ensure proper interaction with tinyMem.\n\n"
	} else {
		// Insert after the IDE Integration heading
		before := readmeContent[:insertionPoint+len("## IDE Integration")]
		after := readmeContent[insertionPoint+len("## IDE Integration"):]

		addition := "\n\n### Agent Setup for MCP Usage\n\n"
		addition += "When using tinyMem as an MCP server for AI agents, ensure that your agents follow the MANDATORY TINYMEM CONTROL PROTOCOL.\n\n"
		addition += "Include the contract content from [AGENT_CONTRACT.md](AGENT_CONTRACT.md) in your agent's system prompt to ensure proper interaction with tinyMem.\n\n"

		readmeContent = before + addition + after
	}

	err = os.WriteFile("README.md", []byte(readmeContent), 0644)
	if err != nil {
		fmt.Printf("Error writing to README.md: %v\n", err)
		return
	}

	fmt.Println("README.md updated with MCP setup instructions")
}

func main() {
	memory_addContract()
}
