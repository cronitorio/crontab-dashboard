package dashboard

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/cronitorio/cronitor-cli/lib"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var exitFn = os.Exit

var Version = "0.1.0"

var cfgFile string
var userAgent string

// Flags that are either global or used in multiple commands
var apiKey string
var environment string
var debugLog string
var dev bool
var hostname string
var pingApiKey string
var verbose bool
var noStdoutPassthru bool
var users string
var pingApiHostFlag string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "crontab-dashboard",
	Short: shortDescription(Version),
	Long: shortDescription(Version) + `

Self-hosted Crontab Guru Dashboard. See https://github.com/cronitorio/crontab-dashboard.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fatal(err.Error(), 1)
	}
}

var varApiKey = "CRONITOR_API_KEY"
var varEnv = "CRONITOR_ENV"
var varHostname = "CRONITOR_HOSTNAME"
var varLog = "CRONITOR_LOG"
var varPingApiKey = "CRONITOR_PING_API_KEY"
var varPingApiHost = "CRONITOR_PING_API_HOST"
var varExcludeText = "CRONITOR_EXCLUDE_TEXT"
var varConfig = "CRONITOR_CONFIG"
var varDashUsername = "CRONITOR_DASH_USER"
var varDashPassword = "CRONITOR_DASH_PASS"
var varAllowedIPs = "CRONITOR_ALLOWED_IPS"
var varUsers = "CRONITOR_USERS"
var varApiVersion = "CRONITOR_API_VERSION"

func init() {
	userAgent = fmt.Sprintf("CrontabGuruDashboard/%s", Version)
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.Version = Version
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", cfgFile, "Config file")
	RootCmd.PersistentFlags().StringVar(&environment, "env", environment, "Cronitor Environment")
	RootCmd.PersistentFlags().StringVarP(&apiKey, "api-key", "k", apiKey, "Cronitor API Key (appears in shell history and process lists; prefer CRONITOR_API_KEY)")
	RootCmd.PersistentFlags().StringVarP(&pingApiKey, "ping-api-key", "p", pingApiKey, "Ping API Key (appears in shell history and process lists; prefer CRONITOR_PING_API_KEY)")
	RootCmd.PersistentFlags().StringVar(&pingApiHostFlag, "ping-api-host", pingApiHostFlag, "Telemetry host for pings, e.g. eu.cronitor.link (default: cronitor.link)")
	RootCmd.PersistentFlags().StringVarP(&hostname, "hostname", "n", hostname, "A unique identifier for this host (default: system hostname)")
	RootCmd.PersistentFlags().StringVarP(&debugLog, "log", "l", debugLog, "Write debug logs to supplied file")
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", verbose, "Verbose output")
	RootCmd.PersistentFlags().StringVarP(&users, "users", "u", users, "Comma-separated list of users whose crontabs to include (default: current user only)")

	RootCmd.PersistentFlags().String("api-version", "", "Cronitor API version (e.g. 2025-11-28)")
	RootCmd.PersistentFlags().BoolVar(&dev, "use-dev", dev, "Dev mode")
	RootCmd.PersistentFlags().MarkHidden("use-dev")

	// Bind flags to viper
	viper.BindPFlag(varApiKey, RootCmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag(varEnv, RootCmd.PersistentFlags().Lookup("env"))
	viper.BindPFlag(varHostname, RootCmd.PersistentFlags().Lookup("hostname"))
	viper.BindPFlag(varLog, RootCmd.PersistentFlags().Lookup("log"))
	viper.BindPFlag(varPingApiKey, RootCmd.PersistentFlags().Lookup("ping-api-key"))
	viper.BindPFlag(varPingApiHost, RootCmd.PersistentFlags().Lookup("ping-api-host"))
	viper.BindPFlag(varConfig, RootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag(varApiVersion, RootCmd.PersistentFlags().Lookup("api-version"))
	viper.BindPFlag(varDashUsername, RootCmd.PersistentFlags().Lookup("dash-username"))
	viper.BindPFlag(varDashPassword, RootCmd.PersistentFlags().Lookup("dash-password"))
	viper.BindPFlag(varUsers, RootCmd.PersistentFlags().Lookup("users"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match
	configFile := viper.GetString(varConfig)

	// If a custom config file is specified by flag or env var, use it. Otherwise use default file.
	if len(configFile) > 0 {
		if len(configFile) < 5 || strings.ToLower(configFile[len(configFile)-5:]) != ".json" {
			fmt.Println("Error: Config file must be a .json file")
		}
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(defaultConfigFileDirectory())
		viper.SetConfigName("cronitor")
	}

	// If a config file is found, read it in. A missing file is normal. A file
	// that exists but cannot be read (permissions, a directory, bad JSON) is
	// reported once so a job does not silently run without its settings.
	if err := viper.ReadInConfig(); err == nil {
		log("Reading config from " + viper.ConfigFileUsed())
	} else if !configFileMissing(err) {
		path := configFile
		if path == "" {
			path = configFilePath()
		}
		fmt.Fprintf(os.Stderr, "Warning: could not read config file %s (%s)\n", path, configReadFailureReason(err))
	}
}

// configReadFailureReason classifies a config read error without repeating
// the parser's diagnostic, which for some formats quotes the offending line
// and could echo a stored secret.
func configReadFailureReason(err error) string {
	var pathErr *fs.PathError
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "permission denied"
	case errors.As(err, &pathErr):
		return "file could not be opened"
	default:
		return "file is not valid JSON"
	}
}

func configFileMissing(err error) bool {
	var notFound viper.ConfigFileNotFoundError
	if errors.As(err, &notFound) {
		return true
	}
	return errors.Is(err, fs.ErrNotExist)
}

func effectiveHostname() string {
	if len(viper.GetString(varHostname)) > 0 {
		return viper.GetString(varHostname)
	}

	hostname, _ := os.Hostname()
	return hostname
}

func effectiveTimezoneLocationName() lib.TimezoneLocationName {

	if runtime.GOOS == "windows" {
		out, err := exec.Command("powershell", "-NoProfile", "-c", "(Get-TimeZone | Select-Object -First 1 -Property Id).Id | Write-Output").CombinedOutput()
		if err == nil {
			return lib.TimezoneLocationName{Name: strings.TrimSpace(fmt.Sprintf("%s", out))}
		}
	}

	// First, check if a TZ or CRON_TZ environemnt variable is set -- Diff var used by diff distros
	if locale, isSetFlag := os.LookupEnv("TZ"); isSetFlag {
		return lib.TimezoneLocationName{Name: locale}
	}

	if locale, isSetFlag := os.LookupEnv("CRON_TZ"); isSetFlag {
		return lib.TimezoneLocationName{Name: locale}
	}

	// Attempt to parse timedatectl (should work on FreeBSD, many linux distros)
	if output, err := exec.Command("timedatectl").Output(); err == nil {
		outputString := strings.Replace(string(output), "Time zone", "Timezone", -1)
		r := regexp.MustCompile(`(?m:Timezone:\s+(\S+).+$)`)
		if ret := r.FindStringSubmatch(outputString); ret != nil && len(ret) > 1 {
			return lib.TimezoneLocationName{Name: ret[1]}
		}
	}

	// If /etc/localtime is a symlink, check what it is linking to
	if localtimeFile, err := os.Lstat("/etc/localtime"); err == nil && localtimeFile.Mode()&os.ModeSymlink == os.ModeSymlink {
		if symlink, _ := os.Readlink("/etc/localtime"); len(symlink) > 0 {
			if strings.Contains(symlink, "UTC") {
				return lib.TimezoneLocationName{Name: "UTC"}
			}

			symlinkParts := strings.Split(symlink, "/")
			return lib.TimezoneLocationName{Name: strings.Join(symlinkParts[len(symlinkParts)-2:], "/")}
		}
	}

	// If we happen to have an /etc/timezone, no guarantee it's used, but read that
	if locale, err := ioutil.ReadFile("/etc/timezone"); err == nil {
		return lib.TimezoneLocationName{Name: string(locale)}
	}

	return lib.TimezoneLocationName{Name: ""}
}

func defaultConfigFileDirectory() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s\\ProgramData\\Cronitor", os.Getenv("SYSTEMDRIVE"))
	}

	return "/etc/cronitor"
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}

	return s[:length]
}

func isPathToDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fileInfo.Mode().IsDir()
}

func log(msg string) {
	msg = redactSecrets(msg)
	debugLog := viper.GetString(varLog)
	if len(debugLog) > 0 {
		f, _ := os.OpenFile(debugLog, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		defer f.Close()
		f.WriteString(msg + "\n")
	}

	if verbose {
		fmt.Println(msg)
	}
}

func fatal(msg string, exitCode int) {
	msg = redactSecrets(msg)
	debugLog := viper.GetString(varLog)
	if len(debugLog) > 0 {
		f, _ := os.OpenFile(debugLog, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		defer f.Close()
		f.WriteString(msg + "\n")
	}

	fmt.Fprintln(os.Stderr, msg)
	exitFn(exitCode)
}

// redactSecrets removes remembered secrets and the configured API / ping keys
// from log lines, verbose output, and errors.
func redactSecrets(msg string) string {

	for _, secret := range []string{
		viper.GetString(varApiKey),
		viper.GetString(varPingApiKey),
	} {
		if secret != "" {
			msg = strings.ReplaceAll(msg, secret, "[REDACTED]")
		}
	}
	return msg
}

func shortDescription(version string) string {
	return fmt.Sprintf("Crontab Guru Dashboard version %s", version)
}

func getCronitorApi() *lib.CronitorApi {
	return &lib.CronitorApi{
		IsDev:          dev,
		IsAutoDiscover: false,
		ApiKey:         viper.GetString(varApiKey),
		UserAgent:      userAgent,
		Logger:         log,
	}
}

func configFilePath() string {
	// First check if there's a config file path from viper (which includes env vars and flags)
	if configPath := viper.GetString(varConfig); len(configPath) > 0 {
		return configPath
	}

	// Fall back to default location if no custom path specified
	return fmt.Sprintf("%s/cronitor.json", defaultConfigFileDirectory())
}

// parseUsers parses the CRONITOR_USERS config value into a slice of usernames
func parseUsers() []string {
	usersConfig := viper.GetString(varUsers)
	if usersConfig == "" {
		return []string{} // Return empty slice for default behavior
	}

	// Split by comma and clean up each username
	users := strings.Split(usersConfig, ",")
	var cleanUsers []string
	for _, user := range users {
		user = strings.TrimSpace(user)
		if user != "" {
			cleanUsers = append(cleanUsers, user)
		}
	}

	return cleanUsers
}

const varAuthManaged = "CRONITOR_AUTH_MANAGED"
const varMachineCredentialName = "CRONITOR_MACHINE_CREDENTIAL_NAME"
