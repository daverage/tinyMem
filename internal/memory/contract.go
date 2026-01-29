package memory

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	fmt.Println("Source of truth: docs/agents/AGENT_CONTRACT.md")
	fmt.Println()
	fmt.Println("It will:")
	fmt.Println("1. Replace the contract in existing files (with a warning).")
	fmt.Println("2. Append the contract to existing files that don't have it.")
	fmt.Println("3. Create missing files in docs/agents/.")
	fmt.Println()

	var response string
	fmt.Print("Are you happy to proceed? (yes/no): ")
	fmt.Scanln(&response)

	if strings.ToLower(response) != "yes" && strings.ToLower(response) != "y" {
		fmt.Println("Operation cancelled.")
		return nil
	}

	files := []string{"AGENTS.md", "QWEN.md", "GEMINI.md", "CLAUDE.md"}
	contractContent, err := getContractContent()
	if err != nil {
		return fmt.Errorf("error fetching contract content: %w", err)
	}

	// Ensure docs/agents directory exists
	agentsDir := filepath.Join("docs", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", agentsDir, err)
	}

	// Check root and docs/agents
	locations := []string{".", agentsDir}

	for _, dir := range locations {
		for _, filename := range files {
			targetPath := filepath.Join(dir, filename)
			if _, err := os.Stat(targetPath); err == nil {
				// File exists, update/append contract
				if err := updateContractInFile(targetPath, contractContent); err != nil {
					return fmt.Errorf("error updating %s: %w", targetPath, err)
				}
			} else if dir == agentsDir {
				// File doesn't exist but it's in the primary agents directory, create it
				if err := createFileWithContract(targetPath, contractContent); err != nil {
					return fmt.Errorf("error creating %s: %w", targetPath, err)
				}
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

func getContractContent() (string, error) {
	// 1. Try local file first (Primary source of truth)
	localPath := filepath.Join("docs", "agents", "AGENT_CONTRACT.md")
	if data, err := os.ReadFile(localPath); err == nil {
		fmt.Printf("Using local contract from %s\n", localPath)
		return string(data), nil
	}

	// 2. Fall back to GitHub (Remote fallback)
	url := "https://raw.githubusercontent.com/daverage/tinyMem/refs/heads/main/docs/agents/AGENT_CONTRACT.md"
	fmt.Printf("Local contract not found at %s, fetching from %s...\n", localPath, url)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch contract: status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func updateContractInFile(filename, contractContent string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	fileContent := string(content)

	// Marker for the start of the contract
	marker := "# TINYMEM CONTROL PROTOCOL"
	idx := strings.Index(fileContent, marker)

	if idx != -1 {
		fmt.Printf("WARNING: Existing contract found in %s. Replacing with latest version.\n", filename)
		// Remove old contract (from marker to end)
		fileContent = fileContent[:idx]
	} else {
		fmt.Printf("Appending contract to %s...\n", filename)
		// Ensure there's a newline if we're appending to a non-empty file
		if len(fileContent) > 0 && !strings.HasSuffix(fileContent, "\n") {
			fileContent += "\n"
		}
	}

	newContent := strings.TrimSpace(fileContent) + "\n\n" + strings.TrimSpace(contractContent) + "\n"
	err = os.WriteFile(filename, []byte(newContent), 0644)
	if err != nil {
		return err
	}

	return nil
}

func createFileWithContract(filename, contractContent string) error {
	// For new files, we just write the contract content directly
	err := os.WriteFile(filename, []byte(strings.TrimSpace(contractContent)+"\n"), 0644)
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
	if strings.Contains(readmeContent, "docs/agents/AGENT_CONTRACT.md") {
		fmt.Println("Contract reference already exists in README.md, skipping.")
		return nil
	}

	// Find a good place to insert the MCP setup instructions
	insertionPoint := strings.Index(readmeContent, "## 🔌 Integration")
	if insertionPoint == -1 {
		// If we can't find the Integration section, append to end
		readmeContent += "\n\n## Setting Up Agents for MCP Usage\n\n"
		readmeContent += "When using tinyMem as an MCP server for AI agents, ensure that your agents follow the MANDATORY TINYMEM CONTROL PROTOCOL.\n\n"
		readmeContent += "Include the contract content from [docs/agents/AGENT_CONTRACT.md](docs/agents/AGENT_CONTRACT.md) in your agent's system prompt to ensure proper interaction with tinyMem.\n\n"
	} else {
		// Insert after the Integration heading
		before := readmeContent[:insertionPoint+len("## 🔌 Integration")]
		after := readmeContent[insertionPoint+len("## 🔌 Integration"):]

		addition := "\n\n### Agent Setup for MCP Usage\n\n"
		addition += "When using tinyMem as an MCP server for AI agents, ensure that your agents follow the MANDATORY TINYMEM CONTROL PROTOCOL.\n\n"
		addition += "Include the contract content from [docs/agents/AGENT_CONTRACT.md](docs/agents/AGENT_CONTRACT.md) in your agent's system prompt to ensure proper interaction with tinyMem.\n\n"

		readmeContent = before + addition + after
	}

	err = os.WriteFile("README.md", []byte(readmeContent), 0644)
	if err != nil {
		return err
	}

	fmt.Println("README.md updated with MCP setup instructions")
	return nil
}
