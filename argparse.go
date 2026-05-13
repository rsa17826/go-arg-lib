package argparse

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

type ArgumentData struct {
	Keys        []string
	AfterCount  int
	Target      any
	Description string
	VarArgs     bool
	AllowDupes  bool
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
				filter := args[i+1:]
				PrintHelp(argDefinitions, filter)
				os.Exit(0)
			}
		}
	}

	// 3. Parsing Logic
	for i := 0; i < len(args); {
		found := false
		for _, def := range argDefinitions {
			if !matches(def.Keys, args[i]) {
				continue
			}
			found = true

			if def.VarArgs {
				// Collect at least AfterCount values, then keep consuming
				// tokens that don't look like a registered flag.
				values, newI := collectVarArgs(args, i+1, def.AfterCount, argDefinitions)
				if def.AllowDupes {
					// Append this invocation's values to whatever was there before.
					if t, ok := def.Target.(*[]string); ok {
						*t = append(*t, values...)
					}
				} else {
					if t, ok := def.Target.(*[]string); ok {
						*t = values
					}
				}
				i = newI
			} else if def.AllowDupes {
				if def.AfterCount == 0 {
					// No value expected — count occurrences.
					if t, ok := def.Target.(*int); ok {
						*t++
					}
					i++
				} else if def.AfterCount == 1 && i+1 < len(args) {
					if t, ok := def.Target.(*[]string); ok {
						*t = append(*t, args[i+1])
					}
					i += 2
				} else if def.AfterCount > 1 && i+def.AfterCount < len(args) {
					if t, ok := def.Target.(*[]string); ok {
						*t = append(*t, args[i+1:i+1+def.AfterCount]...)
					}
					i += 1 + def.AfterCount
				} else {
					i++
				}

			} else {
				// Original single-use logic.
				if def.AfterCount == 0 {
					if t, ok := def.Target.(*bool); ok {
						*t = true
					}
					i++
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
				} else {
					i++
				}
			}
			break
		}
		if !found {
			i++
		}
	}
}

// collectVarArgs collects at least minCount tokens starting at args[start],
// then continues collecting as long as the next token is not a registered key.
// Returns the collected slice and the new index into args.
func collectVarArgs(args []string, start, minCount int, allDefs []ArgumentData) ([]string, int) {
	collected := []string{}
	i := start

	// Consume the minimum required values regardless of whether they look like flags.
	for j := 0; j < minCount && i < len(args); j++ {
		collected = append(collected, args[i])
		i++
	}

	// Consume additional tokens until we hit something that matches a known key.
	for i < len(args) && !isAnyKey(allDefs, args[i]) {
		collected = append(collected, args[i])
		i++
	}

	return collected, i
}

// isAnyKey reports whether token matches a key in any of the provided definitions.
func isAnyKey(defs []ArgumentData, token string) bool {
	for _, def := range defs {
		if matches(def.Keys, token) {
			return true
		}
	}
	return false
}

func PrintHelp(defs []ArgumentData, filters []string) {
	fmt.Println("Usage Options:")
	for _, def := range defs {
		keysStr := "-" + strings.Join(def.Keys, ", -")

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
	return slices.Contains(keys, clean)
}
