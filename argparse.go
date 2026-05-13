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
	ExampleArgs []string
	Default     []any
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

const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Gray   = "\033[90m"
)

func PrintHelp(defs []ArgumentData, filters []string) {
	fmt.Printf("%sUsage Options:%s\n", Bold, Reset)

	// Define column widths
	const (
		col1Width = 20
		col2Width = 30
	)

	// Headers
	fmt.Printf("  %-*s %-*s %s\n", col1Width, "FLAGS", col2Width, "EXPECTS", "DESCRIPTION")
	fmt.Printf("  %s%s\n%s", Gray, strings.Repeat("-", col1Width+col2Width+13), Reset)

	for _, def := range defs {
		// 1. Build Key String (with colors)
		keysFormatted := []string{}
		rawKeysLen := 0
		for i, k := range def.Keys {
			p := "-"
			if len(k) > 1 {
				p = "--"
			}
			keysFormatted = append(keysFormatted, Cyan+p+k+Reset)
			rawKeysLen += len(p) + len(k)
			if i < len(def.Keys)-1 {
				rawKeysLen += 2 // for ", "
			}
		}
		keysStr := strings.Join(keysFormatted, ", ")

		// 2. Build Expects String (with colors)
		var expects, rawExpects string
		if def.VarArgs {
			expects, rawExpects = Yellow+"<val1>...<valN>"+Reset, "<val1>...<valN>"
		} else if def.AfterCount == 0 {
			expects, rawExpects = Blue+"[flag]"+Reset, "[flag]"
		} else {
			noDataCounter := 0
			for i := range def.AfterCount {
				name := "value"
				var default_ any = nil
				var dataFound = false
				if i > 0 {
					expects += " "
					rawExpects += " "
				}
				if i < len(def.ExampleArgs) {
					dataFound = true
					if noDataCounter > 0 {
						if noDataCounter == 1 {
							expects += Yellow + "<1 value>" + Reset
							rawExpects += "<1 value>"
						} else {
							expects += fmt.Sprintf("%s<%d values>%s", Yellow, noDataCounter, Reset)
							rawExpects += fmt.Sprintf("<%d values>", noDataCounter)
							noDataCounter = 0
						}
					}
					name = def.ExampleArgs[i]
				}
				if i < len(def.Default) {
					dataFound = true
					default_ = def.Default[i]
				}
				if dataFound {
					if default_ != nil {
						expects += formatArg(""+name+"", default_)
						rawExpects += "<" + name + "=" + repr(default_) + ">"
					} else {
						expects += formatArg(""+name+"", nil)
						rawExpects += "<" + name + ">"
					}
				} else {
					noDataCounter += 1
				}
			}

		}
		//  else {
		// 	// rawExpects = fmt.Sprintf("<%d values>", def.AfterCount)
		// 	// expects = Yellow + rawExpects + Reset
		// }

		// 3. Filtering logic
		if len(filters) > 0 {
			match := false
			fullKeyMatch := strings.Join(def.Keys, " ")
			for _, f := range filters {
				if strings.Contains(fullKeyMatch, f) || strings.Contains(strings.ToLower(def.Description), strings.ToLower(f)) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
		}

		// 4. PRINTING WITH MANUAL PADDING
		// We subtract the "invisible" color bytes by calculating the difference
		// between the colored string length and the plain text length.
		kPadding := col1Width + (len(keysStr) - rawKeysLen)
		ePadding := col2Width + (len(expects) - len(rawExpects))

		fmt.Printf("  %-*s %-*s %s\n", kPadding, keysStr, ePadding, expects, def.Description)

		// 5. Example Row
		if def.AllowDupes || def.VarArgs || def.AfterCount > 1 || len(def.ExampleArgs) > 0 || len(def.Default) > 0 {
			fmt.Printf("    %sExample:\n    %s%s\n", Gray, formatExample(def), Reset)
		}
	}
}

func repr(v any) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%#v", v)
	// switch val := v.(type) {
	// case string:
	// 	return `"` + val + `"` // Wraps strings in quotes
	// default:
	// 	return fmt.Sprintf("%v", val)
	// }
}
func formatArg(name string, default_ any) string {
	if default_ != nil {
		return Yellow + "<" + Cyan + name + Yellow + "=" + Blue + repr(default_) + Yellow + ">" + Reset
	}
	return Yellow + "<" + Cyan + name + Yellow + ">" + Reset
}

// formatExample generates a dummy usage string based on the definition
func formatExample(def ArgumentData) string {
	p := "-"
	if len(def.Keys[0]) > 1 {
		p = "--"
	}
	rawKey := p + def.Keys[0]
	// Blue for the flag, then Gray for the surrounding "Example:" text
	coloredKey := Blue + rawKey + Gray

	// Helper to get either the custom ExampleArg or the fallback "valN"
	// and wrap it in Yellow + brackets
	getValName := func(index int) string {
		name := fmt.Sprintf("val%d", index+1)
		var default_ any = nil
		if index < len(def.ExampleArgs) {
			name = def.ExampleArgs[index]
		}
		if index < len(def.Default) {
			default_ = def.Default[index]
		}
		if default_ != nil {
			return Blue + repr(default_) + Reset
		}
		return Yellow + "<" + Cyan + name + Yellow + ">" + Reset
	}

	if def.VarArgs {
		v1, v2, v3 := getValName(0), getValName(1), getValName(2)
		if def.AllowDupes {
			return fmt.Sprintf("%s %s %s %s\n    %s %s %s %s %s",
				coloredKey, v1, v2, v3,
				coloredKey, v1, v2, coloredKey, v3)
		}
		return fmt.Sprintf("%s %s %s %s", coloredKey, v1, v2, v3)
	}

	if def.AllowDupes && def.AfterCount == 0 {
		return fmt.Sprintf("%s %s %s (counts occurrences)",
			coloredKey, coloredKey, coloredKey)
	}

	if def.AllowDupes && def.AfterCount == 1 {
		v1, v2 := getValName(0), getValName(1)
		return fmt.Sprintf("%s %s %s %s", coloredKey, v1, coloredKey, v2)
	} else if def.AfterCount == 1 {
		v1 := getValName(0)
		return fmt.Sprintf("%s %s", coloredKey, v1)
	}

	if def.AfterCount > 1 {
		vals := []string{}
		for i := 0; i < def.AfterCount; i++ {
			vals = append(vals, getValName(i))
		}
		return fmt.Sprintf("%s %s", coloredKey, strings.Join(vals, " "))
	}
	return ""
}

func matches(keys []string, input string) bool {
	clean := strings.TrimLeft(input, "-")
	return slices.Contains(keys, clean)
}
