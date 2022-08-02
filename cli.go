package cli

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"text/tabwriter"
)

// CLI is main CLI application definition. It has a name, description, author (which are used only when printing usage syntax), commands and pointers to File instances to which standard output or errors are printed (named respectively stdout and stderr).
type CLI struct {
	name        string
	desc        string
	author      string
	cmds        map[string]*CLICmd
	parsedFlags map[string]string
	parsedArgs  map[string]string
	stdout      *os.File
	stderr      *os.File
	stdin       *os.File
}

// AttachCmd attaches instance of CLICmd to CLI.
func (c *CLI) AttachCmd(cmd *CLICmd) {
	n := cmd.name
	if c.cmds == nil {
		c.cmds = make(map[string]*CLICmd)
	}
	c.cmds[n] = cmd
}

// GetCmd returns instance of CLICmd of command k.
func (c *CLI) GetCmd(k string) *CLICmd {
	return c.cmds[k]
}

// GetSortedCmds returns sorted list of command names.
func (c *CLI) GetSortedCmds() []string {
	cmds := reflect.ValueOf(c.cmds).MapKeys()
	scmds := make([]string, len(cmds))
	for i, cmd := range cmds {
		scmds[i] = cmd.String()
	}
	sort.Strings(scmds)
	return scmds
}

// PrintHelp prints usage info to stdout file.
func (c *CLI) PrintHelp() {
	fmt.Fprintf(c.stdout, c.name+" by "+c.author+"\n"+c.desc+"\n\n")
	fmt.Fprintf(c.stdout, "Usage: "+path.Base(os.Args[0])+" [FLAGS] COMMAND\n\n")
	fmt.Fprintf(c.stdout, "Commands:\n")

	w := new(tabwriter.Writer)
	w.Init(c.stdout, 8, 8, 0, '\t', 0)
	for _, n := range c.GetSortedCmds() {
		cmd := c.GetCmd(n)
		fmt.Fprintf(w, "  %s\t%s\n", n, cmd.desc)
	}
	w.Flush()

	fmt.Fprintf(c.stdout, "\nRun '"+path.Base(os.Args[0])+" COMMAND --help' for more information on a command.\n")
}

// PrintInvalidCmd prints invalid command error to stderr file.
func (c *CLI) PrintInvalidCmd(cmd string) {
	fmt.Fprintf(c.stderr, "Invalid command: "+cmd+"\n\n")
	c.PrintHelp()
}

// AddCmd creates a new command with name n, description d and handler of f. It creates instance of CLICmd, attaches it to CLI and returns it.
func (c *CLI) AddCmd(n string, d string, f func(cli *CLI) int) *CLICmd {
	cmd := NewCLICmd(n, d, f)
	c.AttachCmd(cmd)
	return cmd
}

// AddFlagToCmds adds a flag to all attached commands. It creates CLIFlag instance and attaches it.
func (c *CLI) AddFlagToCmds(n string, a string, hv string, d string, nf int32, fn func(*CLICmd)) {
	for _, n := range c.GetSortedCmds() {
		cmd := c.GetCmd(n)
		flg := NewCLIFlag(n, a, hv, d, nf, fn)
		cmd.AttachFlag(flg)
	}
}

// AddArg adds an argument to all attached commands.
func (c *CLI) AddArgToCmds(n string, hv string, d string, nf int32) {
	for _, n := range c.GetSortedCmds() {
		cmd := c.GetCmd(n)
		if cmd.argsIdx > 9 {
			log.Fatal("Only 10 arguments are allowed")
		}
		arg := NewCLIFlag(n, "", hv, d, nf, nil)
		cmd.AttachArg(arg)
	}
}

// getFlagSetPtrs creates flagset instance, parses flags and returns list of pointers to results of parsing the flags.
func (c *CLI) getFlagSetPtrs(cmd *CLICmd) (map[string]interface{}, map[string]interface{}, []string) {
	fset := flag.NewFlagSet("flagset", flag.ContinueOnError)
	// nothing should come out of flagset
	fset.Usage = func() {}
	fset.SetOutput(ioutil.Discard)

	nptrs := make(map[string]interface{})
	aptrs := make(map[string]interface{})
	fs := cmd.GetSortedFlags()
	for _, n := range fs {
		f := cmd.GetFlag(n)
		if f.IsRequireValue() {
			nptrs[n] = fset.String(n, "", "")
			aptrs[f.alias] = fset.String(f.alias, "", "")
		} else if f.nflags&TypeBool > 0 {
			nptrs[n] = fset.Bool(n, false, "")
			aptrs[f.alias] = fset.Bool(f.alias, false, "")
		}
	}
	fset.Parse(os.Args[2:])
	return nptrs, aptrs, fset.Args()
}

// parseFlags iterates over flags and args and validates them. In case of error it prints out to CLI stderr.
func (c *CLI) parseFlags(cmd *CLICmd) int {
	if c.parsedFlags == nil {
		c.parsedFlags = make(map[string]string)
	}

	fs := cmd.GetSortedFlags()
	nptrs, aptrs, args := c.getFlagSetPtrs(cmd)

	for _, n := range fs {
		f := cmd.GetFlag(n)
		a := f.alias

		var nv string
		var av string
		if f.nflags&TypeBool > 0 {
			c.parsedFlags[n] = "false"
			if *(nptrs[n]).(*bool) == true || *(aptrs[a]).(*bool) == true {
				c.parsedFlags[n] = "true"
				if f.fn != nil {
					f.fn(cmd)
				}
			}
			continue
		}

		nv = *(nptrs[n]).(*string)
		av = *(aptrs[a]).(*string)

		err := f.ValidateValue(false, nv, av)
		if err != nil {
			fmt.Fprintf(c.stderr, "ERROR: "+err.Error()+"\n")
			cmd.PrintHelp(c)
			return 1
		}

		c.parsedFlags[n] = av
		if nv != "" {
			c.parsedFlags[n] = nv
		}
	}

	if c.parsedArgs == nil {
		c.parsedArgs = make(map[string]string)
	}

	as := cmd.GetSortedArgs()

	for i, n := range as {
		v := ""
		if len(args) >= i+1 {
			v = args[i]
		}

		f := cmd.GetArg(n)

		err := f.ValidateValue(true, v, "")
		if err != nil {
			fmt.Fprintf(c.stderr, "ERROR: "+err.Error()+"\n")
			cmd.PrintHelp(c)
			return 1
		}

		c.parsedArgs[n] = v
	}

	postv := cmd.GetPostValidation()
	if postv != nil {
		err := postv(c)
		if err != nil {
			fmt.Fprintf(c.stderr, "ERROR: "+err.Error()+"\n")
			cmd.PrintHelp(c)
			return 1
		}
	}
	return 0
}

// SetStdin sets stdin
func (c *CLI) SetStdin(stdin *os.File) {
	c.stdin = stdin
}

// Run parses the arguments, validates them and executes command handler. In case of invalid arguments, error is printed to stderr and 1 is returned. Return value behaves like exit code.
func (c *CLI) Run(stdout *os.File, stderr *os.File) int {
	c.stdout = stdout
	c.stderr = stderr
	// display help
	if len(os.Args[1:]) < 1 || (len(os.Args[1:]) == 1 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
		c.PrintHelp()
		return 0
	}
	for _, n := range c.GetSortedCmds() {
		if n == os.Args[1] {
			// display command help
			if len(os.Args[1:]) == 2 && (os.Args[2] == "-h" || os.Args[2] == "--help") {
				c.GetCmd(n).PrintHelp(c)
				return 0
			}
			exitCode := c.parseFlags(c.GetCmd(n))
			if exitCode > 0 {
				return exitCode
			}
			return c.GetCmd(n).Run(c)
		}
	}
	// command not found
	c.PrintInvalidCmd(os.Args[1])
	return 1
}

// Flag returns value of flag.
func (c *CLI) Flag(n string) string {
	return c.parsedFlags[n]
}

// Arg returns value of arg.
func (c *CLI) Arg(n string) string {
	return c.parsedArgs[n]
}

// NewCLI creates new instance of CLI with name n, description d and author a and returns it.
func NewCLI(n string, d string, a string) *CLI {
	c := &CLI{name: n, desc: d, author: a}
	return c
}
