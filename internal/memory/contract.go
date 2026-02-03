package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const contractRemoteBase = "https://raw.githubusercontent.com/daverage/tinyMem/refs/heads/main/docs/agents"

// ContractType determines which agent contract file to apply.
type ContractType string

const (
	ContractTypeLarge ContractType = "large"
	ContractTypeSmall ContractType = "small"
)

func (c ContractType) fileName() string {
	switch c {
	case ContractTypeSmall:
		return "AGENT_CONTRACT_SMALL.md"
	default:
		return "AGENT_CONTRACT.md"
	}
}

func (c ContractType) docLink() string {
	return filepath.ToSlash(filepath.Join("docs", "agents", c.fileName()))
}

// EnsureProjectContracts writes the appropriate agent contract into the project files.
func EnsureProjectContracts(projectRoot string, contractType ContractType) error {
	if contractType == "" {
		contractType = ContractTypeLarge
	}

	fmt.Printf("Applying %s agent contract to project files...\n", contractType)
	contractContent, err := getContractContent(projectRoot, contractType)
	if err != nil {
		return fmt.Errorf("error fetching contract content: %w", err)
	}

	agentsDir := filepath.Join(projectRoot, "docs", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("error creating directory %s: %w", agentsDir, err)
	}
	if err := ensureAgentContractFiles(projectRoot); err != nil {
		return fmt.Errorf("error ensuring agent contract files: %w", err)
	}

	files := []string{"AGENTS.md", "QWEN.md", "GEMINI.md", "CLAUDE.md", "CODEX.md"}
	locations := []string{projectRoot, agentsDir}
	for _, dir := range locations {
		for _, filename := range files {
			targetPath := filepath.Join(dir, filename)
			if _, err := os.Stat(targetPath); err == nil {
				if err := updateContractInFile(targetPath, contractContent); err != nil {
					return fmt.Errorf("error updating %s: %w", targetPath, err)
				}
			} else if dir == agentsDir {
				if err := createFileWithContract(targetPath, contractContent); err != nil {
					return fmt.Errorf("error creating %s: %w", targetPath, err)
				}
			}
		}
	}

	jsonTargets := []string{
		filepath.Join(projectRoot, ".qwen", "settings.json"),
		filepath.Join(projectRoot, ".gemini", "settings.json"),
		filepath.Join(projectRoot, ".codex", "settings.json"),
	}
	for _, targetPath := range jsonTargets {
		if _, err := os.Stat(targetPath); err == nil {
			if err := updateContractInJson(targetPath, contractContent); err != nil {
				fmt.Printf("Warning: failed to update JSON file %s: %v\n", targetPath, err)
			}
		}
	}

	fmt.Println("Agent contracts synchronized.")
	return nil
}

func updateContractInJson(filename, contractContent string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		fileContent := string(content)
		if strings.Contains(fileContent, "# TINYMEM CONTROL PROTOCOL") {
			return updateContractInFile(filename, contractContent)
		}
		return fmt.Errorf("file is not a valid JSON object")
	}

	fmt.Printf("Updating JSON contract slot in %s...\n", filename)
	data["tinymem_protocol"] = strings.TrimSpace(contractContent)

	newContent, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, newContent, 0644)
}

func getContractContent(projectRoot string, contractType ContractType) (string, error) {
	localPath := filepath.Join(projectRoot, "docs", "agents", contractType.fileName())
	if data, err := os.ReadFile(localPath); err == nil {
		fmt.Printf("Using local contract from %s\n", localPath)
		return string(data), nil
	}

	remoteURL := fmt.Sprintf("%s/%s", contractRemoteBase, contractType.fileName())
	fmt.Printf("Local contract not found at %s, fetching from %s...\n", localPath, remoteURL)

	resp, err := http.Get(remoteURL)
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
	startMarker := "**Start of tinyMem Protocol**"
	endMarker := "**End of tinyMem Protocol**"

	startIdx := strings.Index(fileContent, startMarker)
	endIdx := strings.Index(fileContent, endMarker)

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		fmt.Printf("Replacing contract block in %s...\n", filename)
		before := fileContent[:startIdx]
		after := fileContent[endIdx+len(endMarker):]
		newContent := before + strings.TrimSpace(contractContent) + after
		return os.WriteFile(filename, []byte(newContent), 0644)
	}

	oldMarker := "# TINYMEM CONTROL PROTOCOL"
	idx := strings.Index(fileContent, oldMarker)

	if idx != -1 {
		fmt.Printf("WARNING: Replacing legacy contract block in %s...\n", filename)
		fileContent = fileContent[:idx]
	} else {
		fmt.Printf("Appending contract to %s...\n", filename)
		if len(fileContent) > 0 && !strings.HasSuffix(fileContent, "\n") {
			fileContent += "\n"
		}
	}

	newContent := strings.TrimSpace(fileContent) + "\n\n" + strings.TrimSpace(contractContent) + "\n"
	if err := os.WriteFile(filename, []byte(newContent), 0644); err != nil {
		return err
	}
	return nil
}

func createFileWithContract(filename, contractContent string) error {
	if err := os.WriteFile(filename, []byte(strings.TrimSpace(contractContent)+"\n"), 0644); err != nil {
		return err
	}
	fmt.Printf("Created %s with contract\n", filename)
	return nil
}

func ensureAgentContractFiles(projectRoot string) error {
	agentsDir := filepath.Join(projectRoot, "docs", "agents")
	contractTypes := []ContractType{ContractTypeLarge, ContractTypeSmall}
	for _, cType := range contractTypes {
		contractPath := filepath.Join(agentsDir, cType.fileName())
		if _, err := os.Stat(contractPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("error checking %s: %w", contractPath, err)
		}

		fmt.Printf("Downloading missing %s contract to %s...\n", cType, contractPath)
		content, err := getContractContent(projectRoot, cType)
		if err != nil {
			return fmt.Errorf("error fetching %s contract: %w", cType, err)
		}

		if err := createFileWithContract(contractPath, content); err != nil {
			return fmt.Errorf("error creating %s: %w", contractPath, err)
		}
	}

	return nil
}

func updateReadme(projectRoot string, contractType ContractType) error {
	readmePath := filepath.Join(projectRoot, "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		return err
	}

	readmeContent := string(content)
	link := contractType.docLink()
	if strings.Contains(readmeContent, link) {
		fmt.Println("Contract reference already exists in README.md, skipping.")
		return nil
	}

	insertionPoint := strings.Index(readmeContent, "## 🔌 Integration")
	addition := fmt.Sprintf("\n\n### Agent Setup for MCP Usage\n\nWhen using tinyMem as an MCP server, ensure your agent references the MANDATORY TINYMEM CONTROL PROTOCOL.\n\nInclude the contract content from [%s](%s) in your agent's system prompt to guarantee alignment with tinyMem.\n\n", link, link)

	if insertionPoint == -1 {
		readmeContent += addition
	} else {
		before := readmeContent[:insertionPoint+len("## 🔌 Integration")]
		after := readmeContent[insertionPoint+len("## 🔌 Integration"):]
		readmeContent = before + addition + after
	}

	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		return err
	}

	fmt.Println("README.md updated with MCP setup instructions")
	return nil
}

// ParseContractType normalizes a string into a ContractType.
func ParseContractType(value string) ContractType {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "small", "tiny", "small-llm":
		return ContractTypeSmall
	default:
		return ContractTypeLarge
	}
}
