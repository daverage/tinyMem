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
	contractContent, err := getContractContent()
	if err != nil {
		return fmt.Errorf("error fetching contract content: %w", err)
	}

	// Ensure docs/agents directory exists
	agentsDir := filepath.Join("docs", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", agentsDir, err)
	}

	for _, filename := range files {
		targetPath := filepath.Join(agentsDir, filename)
		if _, err := os.Stat(targetPath); err == nil {
			// File exists, append contract if not already present
			if err := appendContractToFile(targetPath, contractContent); err != nil {
				return fmt.Errorf("error appending to %s: %w", targetPath, err)
			}
		} else {
			// File doesn't exist, create it with contract
			if err := createFileWithContract(targetPath, contractContent); err != nil {
				return fmt.Errorf("error creating %s: %w", targetPath, err)
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
	// 1. Try local file first
	localPath := filepath.Join("docs", "agents", "AGENT_CONTRACT.md")
	if data, err := os.ReadFile(localPath); err == nil {
		fmt.Printf("Using local contract from %s\n", localPath)
		return string(data), nil
	}

	// 2. Fall back to GitHub
	url := "https://raw.githubusercontent.com/a-marczewski/tinyMem/refs/heads/main/docs/agents/AGENT_CONTRACT.md"
	fmt.Printf("Local contract not found, fetching from %s...\n", url)

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

	newContent := fileContent + "\n" + contractContent
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
