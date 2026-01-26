package memory

import (
	"fmt"
	"os"
	"strings"
)

// AddContract adds the MANDATORY TINYMEM CONTROL PROTOCOL to agent markdown files
func AddContract() error {
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
		return nil
	}
	
	files := []string{"AGENTS.md", "QWEN.md", "GEMINI.md", "CLAUDE.md"}
	contractContent := getContractContent()
	
	for _, filename := range files {
		if _, err := os.Stat(filename); err == nil {
			// File exists, append contract if not already present
			if err := appendContractToFile(filename, contractContent); err != nil {
				return fmt.Errorf("error appending to %s: %w", filename, err)
			}
		} else {
			// File doesn't exist, create it with contract
			if err := createFileWithContract(filename, contractContent); err != nil {
				return fmt.Errorf("error creating %s: %w", filename, err)
			}
		}
	}
	
	// Update README.md
	if err := updateReadme(); err != nil {
		return fmt.Errorf("error updating README.md: %w", err)
	}
	
	fmt.Println("Operation completed successfully!")
	return nil
}

func getContractContent() string {
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
*   **Valid:** \`auth middleware\`, \`SQLite FTS\`, \`proxy port\`, \`CLI flags\`, \`user schema\`
*   **Invalid:** \`implementation\`, \`logic\`, \`details\`, \`code\`, \`help\`

These concepts define your search parameters.

#### Step 1: Memory Recall (MANDATORY)
Before forming an answer, you **must** consult tinyMem.
*   **Action:** Call \`memory_query(query="...")\` using the concepts from Step 0.
*   **Broad Context:** If a project-wide view is needed, call \`memory_query(query="")\`.
*   **Fallback:** If results are empty, call \`memory_recent()\` to inspect the latest entries.

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
*   **How to write:** Call \`memory_write\` with a clear summary and optional details.
*   **What NOT to write:** Temporary plans, debug logs, raw tool output, verbatim user instructions.

#### Step 5: Memory Health (CONDITIONAL)
If the user asks about memory state, connectivity, or scope:
*   **Action:** Call \`memory_health\` before replying.
*   **Constraint:** Never guess the status of the memory database.

---

### 3. COMPLIANCE & ENFORCEMENT

#### Tool Honesty Rule
*   Never fabricate tool outputs.
*   Never imply a tool was called when it was not.
*   All tool calls must occur **before** the final response text is generated.

#### The Enforcement Invariant
For any project-related request:

> **A valid response must be preceded by at least one successful tinyMem memory call (\`memory_query\` or \`memory_recent\`) in the same generation run.**

If this invariant is violated, the response is structurally invalid.

---

### 4. MENTAL MODEL

1.  **TinyMem is the source of continuity.** It bridges the gap between sessions.
2.  **You are the interface.** Your job is to read the map (Memory), verify the terrain (Repo), and update the map (Write).
3.  **Silence is failure.** Falling back to generic training data without checking memory is a protocol violation.
`
}

func appendContractToFile(filename, contractContent string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	
	fileContent := string(content)
	if strings.Contains(fileContent, "MANDATORY TINYMEM CONTROL PROTOCOL") {
		fmt.Printf("Contract already exists in %s, skipping.\n", filename)
		return nil
	}
	
	newContent := fileContent + contractContent
	err = os.WriteFile(filename, []byte(newContent), 0644)
	if err != nil {
		return err
	}
	
	fmt.Printf("Contract appended to %s\n", filename)
	return nil
}

func createFileWithContract(filename, contractContent string) error {
	content := "# Agent Contract for tinyMem\n\n" + contractContent
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}
	
	fmt.Printf("Created %s with contract\n", filename)
	return nil
}

func updateReadme() error {
	content, err := os.ReadFile("README.md")
	if err != nil {
		return err
	}
	
	readmeContent := string(content)
	if strings.Contains(readmeContent, "MANDATORY TINYMEM CONTROL PROTOCOL") {
		fmt.Println("Contract section already exists in README.md, skipping.")
		return nil
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
		before := readmeContent[:insertionPoint + len("## IDE Integration")]
		after := readmeContent[insertionPoint + len("## IDE Integration"):]
		
		addition := "\n\n### Agent Setup for MCP Usage\n\n"
		addition += "When using tinyMem as an MCP server for AI agents, ensure that your agents follow the MANDATORY TINYMEM CONTROL PROTOCOL.\n\n"
		addition += "Include the contract content from [AGENT_CONTRACT.md](AGENT_CONTRACT.md) in your agent's system prompt to ensure proper interaction with tinyMem.\n\n"
		
		readmeContent = before + addition + after
	}
	
	err = os.WriteFile("README.md", []byte(readmeContent), 0644)
	if err != nil {
		return err
	}
	
	fmt.Println("README.md updated with MCP setup instructions")
	return nil
}