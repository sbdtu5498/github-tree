package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

var (
	ownerFlag    string
	repoFlag     string
	pathFlag     string
	maxDepthFlag int
)

type File struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func init() {
	flag.StringVar(&ownerFlag, "O", "", "Repository owner")
	flag.StringVar(&ownerFlag, "owner", "", "Repository owner")

	flag.StringVar(&repoFlag, "R", "", "Repository name")
	flag.StringVar(&repoFlag, "repo", "", "Repository name")

	flag.StringVar(&pathFlag, "P", "", "Path within the repository")
	flag.StringVar(&pathFlag, "path", "", "Path within the repository")

	flag.IntVar(&maxDepthFlag, "M", 1, "Maximum depth for fetching content")
	flag.IntVar(&maxDepthFlag, "maxDepth", 1, "Maximum depth for fetching content")
}

func main() {

	// Parse command-line flags
	flag.Parse()

	// Get the absolute path to github-tree-inputs.txt
	inputsFilePath := getAbsolutePath("github-tree-inputs.txt")

	// Check if github-tree-inputs.txt exists
	_, err := os.Stat(inputsFilePath)

	if err == nil {
		// The file exists, so read existing inputs from the file
		currentOwner, currentRepo, currentPath, currentMaxDepth := readInputsFromFile(inputsFilePath)

		// Check if the owner and repo fields are empty
		if currentOwner == "" || currentRepo == "" {
			panic("The 'owner' and 'repo' fields in github-tree-inputs.txt cannot be empty")
		}

		// Set default value for maxDepth if not available
		if currentMaxDepth < 1 {
			currentMaxDepth = 1
		}

		// Update inputs if flags were provided
		if ownerFlag != "" {
			currentOwner = ownerFlag
		}
		if repoFlag != "" {
			currentRepo = repoFlag
		}
		if pathFlag != "" {
			currentPath = pathFlag
		}
		currentMaxDepth = maxDepthFlag
		

		// Update the inputs in the file
		updateInputsInFile(inputsFilePath, currentOwner, currentRepo, currentPath, currentMaxDepth)

		// Retrieve access token from environment
		accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
		if accessToken == "" {
			panic("GitHub access token not found in environment")
		}

		// Fetch files and folders using the updated inputs
		fetchFilesAndFolders(accessToken, currentOwner, currentRepo, currentPath, "", 1, currentMaxDepth)
	} else {
		// The "github-tree-inputs.txt" doesn't exist, so create it

		// Set default value for maxDepth if not available
		if maxDepthFlag == 0 {
			maxDepthFlag = 1
		}

		// Create a new inputs struct
		newInputs := struct {
			Owner    string `json:"owner"`
			Repo     string `json:"repo"`
			Path     string `json:"path"`
			MaxDepth int    `json:"maxDepth"`
		}{
			Owner:    ownerFlag,
			Repo:     repoFlag,
			Path:     pathFlag,
			MaxDepth: maxDepthFlag,
		}

		// Convert to JSON
		newInputsJSON, err := json.MarshalIndent(newInputs, "", "  ")
		if err != nil {
			panic(fmt.Errorf("failed to marshal new inputs: %w", err))
		}

		// Write to the file
		err = os.WriteFile(inputsFilePath, newInputsJSON, 0644)
		if err != nil {
			panic(fmt.Errorf("failed to write new inputs to file: %w", err))
		}

		// Retrieve access token from environment
		accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
		if accessToken == "" {
			panic("GitHub access token not found in environment")
		}

		// Fetch files and folders using the new inputs
		fetchFilesAndFolders(accessToken, ownerFlag, repoFlag, pathFlag, "", 1, maxDepthFlag)
	}
}

func readInputsFromFile(filePath string) (owner, repo, path string, maxDepth int) {
	// Read the contents of the file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to read inputs from file: %w", err))
	}

	// Unmarshal the JSON data into a struct
	var inputs struct {
		Owner    string `json:"owner"`
		Repo     string `json:"repo"`
		Path     string `json:"path"`
		MaxDepth int    `json:"maxDepth"`
	}
	err = json.Unmarshal(fileData, &inputs)
	if err != nil {
		panic(fmt.Errorf("failed to parse inputs from file: %w", err))
	}

	return inputs.Owner, inputs.Repo, inputs.Path, inputs.MaxDepth
}

func getAbsolutePath(filePath string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failed to get current directory: %w", err))
	}

	return filepath.Join(currentDir, filePath)
}

func fetchFilesAndFolders(accessToken, owner, repo, path, indent string, level, maxDepth int) {
	// Stop if the maximum depth has been reached
	if level > maxDepth {
		return
	}

	// Make the API request
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	// Unmarshal the response into a slice of File structs
	var files []File
	err = json.Unmarshal(body, &files)
	if err != nil {
		panic(err)
	}

	// Iterate over the files and folders
	for i, f := range files {
		isLast := i == len(files)-1
		if f.Type == "file" {
			fmt.Printf("%s%s", indent, getFilePrefix(isLast))
			fmt.Println(f.Name)
		} else if f.Type == "dir" {
			fmt.Printf("%s%s", indent, getDirPrefix(isLast))
			fmt.Println(f.Name)
			// Recursively fetch files and folders for subdirectory
			fetchFilesAndFolders(accessToken, owner, repo, path+"/"+f.Name, indent+getIndentPrefix(isLast), level+1, maxDepth)
		}
	}
}

func getFilePrefix(isLast bool) string {
	if isLast {
		return "└── "
	}
	return "├── "
}

func getDirPrefix(isLast bool) string {
	if isLast {
		return "└── "
	}
	return "├── "
}

func getIndentPrefix(isLast bool) string {
	if isLast {
		return "    "
	}
	return "│   "
}

func updateInputsInFile(filePath, owner, repo, path string, maxDepth int) {
	// Create the new inputs struct
	newInputs := struct {
		Owner    string `json:"owner"`
		Repo     string `json:"repo"`
		Path     string `json:"path"`
		MaxDepth int    `json:"maxDepth"`
	}{
		Owner:    owner,
		Repo:     repo,
		Path:     path,
		MaxDepth: maxDepth,
	}

	// Convert to JSON
	newInputsJSON, err := json.MarshalIndent(newInputs, "", "  ")
	if err != nil {
		panic(fmt.Errorf("failed to marshal new inputs: %w", err))
	}

	// Write to the file
	err = os.WriteFile(filePath, newInputsJSON, 0644)
	if err != nil {
		panic(fmt.Errorf("failed to write updated inputs to file: %w", err))
	}
}
