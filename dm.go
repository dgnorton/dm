package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"strings"
	"sync"
	"text/template"
	"unicode"
	"unicode/utf8"
)

type config struct {
	User string
}

var defaultConfig = &config{
	"", // User
}

func loadConfig() (*config, error) {
	cfgfile, _ := configFile()
	bytes, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		return nil, err
	}
	var cfg config
	err = json.Unmarshal(bytes, &cfg)
	return &cfg, err
}

func saveConfig(c *config) error {
	cfgfile, _ := configFile()
	bytes, err := json.MarshalIndent(c, "", "   ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(cfgfile, bytes, 0600)
	return err
}

// A Command is an implementation of a dm command.
// This is borrowed largely from golang's cmd/go/main.go.
type Command struct {
	Run       func(cfg *config, cmd *Command, args []string)
	UsageLine string
	Short     string
	Long      string
	Flag      flag.FlagSet
}

func (c *Command) Name() string {
	name := c.UsageLine
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Command) Usage() {
	fmt.Fprintf(os.Stderr, "usage: %s\n\n", c.UsageLine)
	fmt.Fprintf(os.Stderr, "%s\n", strings.TrimSpace(c.Long))
	os.Exit(2)
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

// Available commands
var commands = []*Command{
	cmdUser,
	cmdSync,
	cmdFind,
	cmdRm,
}

var exitStatus = 0
var exitMu sync.Mutex

func setExitStatus(n int) {
	exitMu.Lock()
	if exitStatus < n {
		exitStatus = n
	}
	exitMu.Unlock()
}

func main() {
	var mainUser string // -u user name

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.StringVar(&mainUser, "u", "", "")
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		usage()
	}

	if args[0] == "help" {
		help(args[1:])
		return
	}

	// make working directory if it doesn't exist...or die trying
	makeWorkDir()

	cfg, err := loadConfig()
	if err != nil {
		if os.IsNotExist(err) {
			// create a default config file
			err = saveConfig(defaultConfig)
			if err != nil {
				log.Fatalf("%s", err)
			}
			cfg = defaultConfig
		} else {
			log.Fatalf("%s", err)
		}
	}

	// command line flags override config file
	if mainUser != "" {
		cfg.User = mainUser
	}

	if cfg.User == "" {
		log.Fatalf(
			`No user set.  Either use the 'dm user <user name>' command or
the '-u <user name>' command line argument.`)
	}

	for _, cmd := range commands {
		if cmd.Name() == args[0] && cmd.Run != nil {
			cmd.Flag.Usage = func() { cmd.Usage() }
			cmd.Flag.Parse(args[1:])
			args = cmd.Flag.Args()
			cmd.Run(cfg, cmd, args)
			exit()
			return
		}
	}

	fmt.Fprintf(os.Stderr, "dm: unknown subcommand %q\n", args[0])
	setExitStatus(2)
	exit()
}

var atexitFuncs []func()

func atexit(f func()) {
	atexitFuncs = append(atexitFuncs, f)
}

func exit() {
	for _, f := range atexitFuncs {
		f()
	}
	os.Exit(exitStatus)
}

func userDir(user string) (string, error) {
	wrk, err := workDir()
	if err != nil {
		return "", err
	}
	return path.Join(wrk, user), nil
}

func makeUserDir(user string) {
	usr, err := userDir(user)
	if err != nil {
		log.Fatalf("%s", err)
	}
	exists, err := isDir(usr)
	if err != nil {
		log.Fatalf("%s", err)
	}
	if exists == false {
		err = os.MkdirAll(usr, 0700)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}
}

func homeDir() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return usr.HomeDir, nil
}

func workDir() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	return path.Join(home, ".dailymile_cli"), nil
}

func makeWorkDir() {
	wrkdir, err := workDir()
	if err != nil {
		log.Fatalf("%s", err)
	}

	exists, err := isDir(wrkdir)
	if err != nil {
		log.Fatalf("%s", err)
	}

	if exists == false {
		err = os.MkdirAll(wrkdir, 0700)
		if err != nil {
			log.Fatalf("%s", err)
		}
	}
}

func configFile() (string, error) {
	wrkdir, err := workDir()
	if err != nil {
		return "", err
	}
	cfgfile := path.Join(wrkdir, "config")
	return cfgfile, nil
}

func isDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		return fi.IsDir(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func isFile(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		return !fi.IsDir(), nil
	} else if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func errorf(format string, args ...interface{}) {
	log.Printf(format, args...)
	setExitStatus(1)
}

var logf = log.Printf

func usage() {
	fmt.Fprintf(os.Stderr, "dm is a command line tool for Dailymile.com\n\n  usage: dm command [args]\n")
	os.Exit(2)
}

var usageTemplate = `dm is a command line tool for DailyMile.com.

Usage:

        dm [-u user name] command [arguments]

The commands are:
{{range .}}{{if .Runnable}}
    {{.Name | printf "%-11s"}} {{.Short}}{{end}}{{end}}

Use "dm help [topic]" for more information about that topic.

`

var helpTemplate = `{{if .Runnable}}usage: dm {{.UsageLine}}

{{end}}{{.Long | trim}}
`

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToTitle(r)) + s[n:]
}

func help(args []string) {
	if len(args) == 0 {
		printUsage(os.Stdout)
		// not exit 2:  succeeded at 'dm help'.
		return
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: dm help command\n\nToo many arguments given.\n")
		os.Exit(2) // failed at 'dm help'
	}

	arg := args[0]

	for _, cmd := range commands {
		if cmd.Name() == arg {
			tmpl(os.Stdout, helpTemplate, cmd)
			// not exit 2:  succeeded at 'dm help cmd'.
			return
		}
	}

	fmt.Fprintf(os.Stderr, "Unknown help topic %#q.  Run 'dm help'.\n", arg)
	os.Exit(2) // failed at 'dm help cmd'
}

func tmpl(w io.Writer, text string, data interface{}) {
	t := template.New("top")
	t.Funcs(template.FuncMap{"trim": strings.TrimSpace, "capitalize": capitalize})
	template.Must(t.Parse(text))
	if err := t.Execute(w, data); err != nil {
		panic(err)
	}
}

func printUsage(w io.Writer) {
	tmpl(w, usageTemplate, commands)
}
