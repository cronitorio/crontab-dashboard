package dashboard

import (
	"fmt"
	"github.com/cronitorio/cronitor-cli/lib"
	"strings"
)

const maxNameLen = 75

func createDefaultName(line *lib.Line, crontab *lib.Crontab, effectiveHostname string, excludeFromName []string, allNameCandidates map[string]bool) string {
	excludeFromName = append(excludeFromName, "2>&1")
	excludeFromName = append(excludeFromName, "/bin/bash -l -c")
	excludeFromName = append(excludeFromName, "/bin/bash -lc")
	excludeFromName = append(excludeFromName, "/bin/bash -c -l")
	excludeFromName = append(excludeFromName, "/bin/bash -cl")
	excludeFromName = append(excludeFromName, "/dev/null")

	excludeFromName = append(excludeFromName, "'")
	excludeFromName = append(excludeFromName, "\"")
	excludeFromName = append(excludeFromName, "\\")

	// Limit the visible hostname portion to 21 chars
	formattedHostname := ""
	if effectiveHostname != "" {
		if len(effectiveHostname) > 21 {
			effectiveHostname = fmt.Sprintf("%s...%s", effectiveHostname[:9], effectiveHostname[len(effectiveHostname)-9:])
		}
		formattedHostname = fmt.Sprintf("[%s] ", effectiveHostname)
	}

	if line.IsAutoDiscoverCommand() {
		return truncateString(fmt.Sprintf("%sAuto discover %s", formattedHostname, strings.TrimSpace(crontab.DisplayName())), maxNameLen)
	}

	// Remove output redirection
	CommandToRun := line.CommandToRun
	for _, redirectionOperator := range []string{">>", ">"} {
		if operatorPosition := strings.LastIndex(line.CommandToRun, redirectionOperator); operatorPosition > 0 {
			CommandToRun = CommandToRun[:operatorPosition]
			break
		}
	}

	// Remove exclusion text
	for _, substr := range excludeFromName {
		CommandToRun = strings.Replace(CommandToRun, substr, "", -1)
	}

	CommandToRun = strings.Join(strings.Fields(CommandToRun), " ")

	// Assemble the candidate name.
	// Ensure we don't produce dupe names if the user has same command on multiple lines.
	formattedRunAs := ""
	lineNumSuffix := fmt.Sprintf(" L%d", line.LineNumber)
	if line.RunAs != "" {
		formattedRunAs = fmt.Sprintf("%s ", line.RunAs)
	}

	candidate := formattedHostname + formattedRunAs + CommandToRun

	if _, exists := allNameCandidates[candidate]; exists {
		candidate = fmt.Sprintf("%s%s", candidate, lineNumSuffix)
	} else {
		allNameCandidates[candidate] = true
	}

	// Return if short, truncate if necessary.
	if maxNameLen >= len(candidate) {
		return candidate
	}

	// Keep the first and last portion of the command
	separator := "..."

	commandPrefixLen := 20 + len(formattedHostname) + len(formattedRunAs)
	commandSuffixLen := maxNameLen - len(lineNumSuffix) - commandPrefixLen - len(separator)
	return fmt.Sprintf(
		"%s%s%s%s",
		strings.TrimSpace(candidate[:commandPrefixLen]),
		separator,
		strings.TrimSpace(candidate[len(candidate)-commandSuffixLen:]), lineNumSuffix)
}
