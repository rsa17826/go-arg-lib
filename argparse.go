package argparse

import (
	"fmt"
	"os"
	"strings"
)

type ArgumentData struct {
	Keys        []string
	AfterCount  int
	Target      any
	Description string
}

type ParseOptions struct {
	DisableDefaultHelp bool
}

func ParseArgs(argDefinitions []ArgumentData, opts ...ParseOptions) {
	args := os.Args[1:]
	disableHelp := false
	if len(opts) > 0 && opts[0].DisableDefaultHelp {
		disableHelp = true
	}

	// 1. Handle "--" separator
	for i, v := range args {
		if v == "--" {
			args = args[:i]
			break
		}
	}

	// 2. Automatic Help Logic
	if !disableHelp {
		for i, arg := range args {
			if matches([]string{"h", "help"}, arg) {
				// Get filter keywords if they exist: -h test t2
				filter := args[i+1:]
				printHelp(argDefinitions, filter)
				os.Exit(0)
			}
		}
	}

	// 3. Existing Parsing Logic
	for i := 0; i < len(args); {
		found := false
		for _, def := range argDefinitions {
			if matches(def.Keys, args[i]) {
				found = true
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

func printHelp(defs []ArgumentData, filters []string) {
	fmt.Println("Usage Options:")
	for _, def := range defs {
		keysStr := "-" + strings.Join(def.Keys, ", -")

		// If filters are provided, check if any key or description matches
		if len(filters) > 0 {
			match := false
			for _, f := range filters {
				if strings.Contains(keysStr, f) || strings.Contains(strings.ToLower(def.Description), strings.ToLower(f)) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		fmt.Printf("  %-20s %s\n", keysStr, def.Description)
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
