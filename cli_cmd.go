package cli

import (
	"fmt"
	"log"
	"os"
	"path"
	"reflect"
	"sort"
	"text/tabwriter"
)

// CLICmd represent a command which has a name (used in args when calling app), description, a handler and flags attached to it.
type CLICmd struct {
	name           string
	desc           string
	flags          map[string]*CLIFlag
	args           map[string]*CLIFlag
	argsOrder      []string
	argsIdx        int
	handler        func(c *CLI) int
	postValidation func(*CLI) error
}

// GetSortedArgs returns arguments list of arg names sorted how they were added but required ones are first.
func (c *CLICmd) GetSortedArgs() []string {
	a := make([]string, c.argsIdx)
	idx := 0
	for i := 0; i < c.argsIdx; i++ {
		n := c.argsOrder[i]
		f := c.GetArg(n)
		if f.nflags&Required > 0 {
			a[idx] = n
			idx++
		}
	}
	for i := 0; i < c.argsIdx; i++ {
		n := c.argsOrder[i]
		f := c.GetArg(n)
		if f.nflags&Required == 0 {
			a[idx] = n
			idx++
		}
	}
	return a
}

func (c *CLICmd) getArgsHelpLine() string {
	sr := ""
	so := ""
	if c.argsIdx > 0 {
		for i := 0; i < c.argsIdx; i++ {
			n := c.argsOrder[i]
			f := c.GetArg(n)
			if f.nflags&Required > 0 {
				sr += " " + f.helpValue
			} else {
				so += " [" + f.helpValue + "]"
			}
		}
	}
	return sr + so
}

// PrintHelp prints command usage information to stdout file.
func (c *CLICmd) PrintHelp(cli *CLI) {
	fmt.Fprintf(cli.stdout, fmt.Sprintf("\nUsage:  %s %s [FLAGS]%s\n\n", path.Base(os.Args[0]), c.name, c.getArgsHelpLine()))
	fmt.Fprintf(cli.stdout, fmt.Sprintf("%s\n", c.desc))

	w := new(tabwriter.Writer)
	w.Init(cli.stdout, 8, 8, 0, '\t', 0)

	var s [2]string
	i := 1
	for _, n := range c.GetSortedFlags() {
		flag := c.GetFlag(n)
		if flag.nflags&Required > 0 {
			i = 0
		} else {
			i = 1
		}
		s[i] += flag.GetHelpLine()
	}

	if s[0] != "" {
		fmt.Fprintf(w, "\nRequired flags: \n")
		fmt.Fprintf(w, s[0])
		w.Flush()
	}
	if s[1] != "" {
		fmt.Fprintf(w, "\nOptional flags: \n")
		fmt.Fprintf(w, s[1])
		w.Flush()
	}

}

// AttachFlag attaches instance of CLIFlag to CLICmd.
func (c *CLICmd) AttachFlag(flag *CLIFlag) {
	n := flag.name
	if c.flags == nil {
		c.flags = make(map[string]*CLIFlag)
	}
	c.flags[n] = flag
}

// AttachArg attaches instance of CLIFlag to CLICmd but as an argument.
func (c *CLICmd) AttachArg(flag *CLIFlag) {
	n := flag.name
	if c.args == nil {
		c.args = make(map[string]*CLIFlag)
	}
	if c.argsOrder == nil {
		c.argsOrder = make([]string, 10)
	}
	c.args[n] = flag
	c.argsOrder[c.argsIdx] = n
	c.argsIdx++
}

// AddFlag adds a flag to a command. It creates CLIFlag instance and attaches it.
func (c *CLICmd) AddFlag(n string, a string, hv string, d string, nf int32, fn func(*CLICmd)) {
	flg := NewCLIFlag(n, a, hv, d, nf, fn)
	c.AttachFlag(flg)
}

// AddArg adds an argument to a command.
func (c *CLICmd) AddArg(n string, hv string, d string, nf int32) {
	if c.argsIdx > 9 {
		log.Fatal("Only 10 arguments are allowed")
	}
	arg := NewCLIFlag(n, "", hv, d, nf, nil)
	c.AttachArg(arg)
}

// AddPostValidation attaches an additional validation function that is executed after the default CLI validation
func (c *CLICmd) AddPostValidation(fn func(*CLI) error) {
	c.postValidation = fn
}

// GetPostValidation returns post validation function
func (c *CLICmd) GetPostValidation() (fn func(*CLI) error) {
	return c.postValidation
}

// GetFlag returns instance of CLIFlag of flag k.
func (c *CLICmd) GetFlag(k string) *CLIFlag {
	return c.flags[k]
}

// GetArg returns instance of CLIFlag of argument k.
func (c *CLICmd) GetArg(k string) *CLIFlag {
	return c.args[k]
}

// GetSortedFlags returns sorted list of flag names.
func (c *CLICmd) GetSortedFlags() []string {
	fs := reflect.ValueOf(c.flags).MapKeys()
	sfs := make([]string, len(fs))
	for i, f := range fs {
		sfs[i] = f.String()
	}
	sort.Strings(sfs)
	return sfs
}

// GetFlags returns list of flag names.
func (c *CLICmd) GetFlags() []reflect.Value {
	return reflect.ValueOf(c.flags).MapKeys()
}

// Run calls command handler.
func (c *CLICmd) Run(cli *CLI) int {
	return c.handler(cli)
}

// NewCLICmd creates CLICmd instance with name n, description d and handler f and returns it.
func NewCLICmd(n string, d string, f func(cli *CLI) int) *CLICmd {
	c := &CLICmd{name: n, desc: d, handler: f}
	return c
}
