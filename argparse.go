package main

import (
	"os"
	"strings"
)

type ArgumentData struct {
	Keys       []string // Multiple keys like ["silent", "headless"]
	AfterCount int      // How many args to consume after the key
	Target     any      // Default value if not found
}

// Helper to check if a key matches this ArgumentData
func (a ArgumentData) Matches(input string) bool {
	cleanInput := strings.TrimLeft(input, "-")
	for _, k := range a.Keys {
		if k == cleanInput {
			return true
		}
	}
	return false
}

// func CheckArgs(argDefinitions []ArgumentData) []any {
// 	// Initialize results with default values
// 	results := make([]any, len(argDefinitions))
// 	for i, def := range argDefinitions {
// 		results[i] = def.Default
// 	}

// 	// Use os.Args[1:] (skipping the program name)
// 	// We handle the "--" separator logic here
// 	args := os.Args[1:]
// 	for i, v := range args {
// 		if v == "--" {
// 			args = args[:i]
// 			break
// 		}
// 	}

// 	// Manual iteration to allow variable step sizes
// 	for i := 0; i < len(args); {
// 		currentArg := args[i]
// 		found := false

// 		for idx, def := range argDefinitions {
// 			if def.Matches(currentArg) {
// 				found = true

// 				if def.AfterCount == 0 {
// 					results[idx] = true
// 					i += 1 // Move to next arg
// 				} else if def.AfterCount == 1 {
// 					if i+1 < len(args) {
// 						results[idx] = args[i+1]
// 						i += 2 // Consume key and 1 value
// 					} else {
// 						fmt.Printf("err: %s requires 1 arg\n", currentArg)
// 						i += 1
// 					}
// 				} else if def.AfterCount > 1 {
// 					if i+def.AfterCount < len(args) {
// 						results[idx] = args[i+1 : i+1+def.AfterCount]
// 						i += 1 + def.AfterCount // Consume key and N values
// 					} else {
// 						fmt.Printf("err: %s requires %d args\n", currentArg, def.AfterCount)
// 						i = len(args) // Stop processing
// 					}
// 				}
// 				break // Exit the definitions loop
// 			}
// 		}

// 		if !found {
// 			i++ // Skip unknown arg
// 		}
// 	}

// 	return results
// }

func ParseArgs(argDefinitions []ArgumentData) {
	args := os.Args[1:]

	// Handle "--" separator
	for i, v := range args {
		if v == "--" {
			args = args[:i]
			break
		}
	}

	for i := 0; i < len(args); {
		found := false
		for _, def := range argDefinitions {
			if matches(def.Keys, args[i]) {
				found = true

				// Logic to fill the Target pointer based on AfterCount
				if def.AfterCount == 0 {
					if t, ok := def.Target.(*bool); ok {
						*t = true
					}
					i += 1
				} else if def.AfterCount == 1 && i+1 < len(args) {
					if t, ok := def.Target.(*string); ok {
						*t = args[i+1]
					}
					i += 2
				} else if def.AfterCount > 1 && i+def.AfterCount < len(args) {
					if t, ok := def.Target.(*[]string); ok {
						*t = args[i+1 : i+1+def.AfterCount]
					}
					i += 1 + def.AfterCount
				}
				break
			}
		}
		if !found {
			i++
		}
	}
}

func matches(keys []string, input string) bool {
	clean := strings.TrimLeft(input, "-")
	for _, k := range keys {
		if k == clean {
			return true
		}
	}
	return false
}
