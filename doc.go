/*
Package cli is meant to make handling command line interface easier.

You define commands with flags, attach a handler to it and package does all
the parsing.

Install

Ensure you have your workspace directory created.

Example

Let's start with an example covering everything. First, let's create main
CLI instance and commands:

    func main() {
        myCLI      := cli.NewCLI("Example CLI", "Silly app", "Author <a@example.com>")

        cmdInit    := myCLI.AddCmd("init", "Initialises the project", InitHandler)
        cmdStart   := myCLI.AddCmd("start", "Start the application", StartHandler)
    }

Next, let's add flags to our commands (inside main()):

    cmdInit.AddFlag("template", "t", "filepath", "Path to template file", TypePathFile|MustExist|Required, nil)
    cmdInit.AddFlag("file-output", "o", "filepath", "Output to a specific file instead of stdout", TypePathFile, nil)
    cmdInit.AddFlag("number", "n", "int", "Number necessary for initialisation", TypeInt|Required, nil)

    cmdStart.AddFlag("verbose", "v", "", "Verbose mode", TypeBool, nil)
    cmdStart.AddFlag("username", "u", "username", "Username", TypeAlphanumeric|AllowDots|AllowUnderscore|Required, nil)
    cmdStart.AddFlag("threshold", "", "1.5", "Threshold, default 1.5", TypeFloat, nil)
    cmdStart.AddArg("input", "FILE", "Path to a file", TypePathFile|Required)
    cmdStart.AddArg("difficulty", "DIFFICULTY", "Level of difficulty (1-5), default 3", TypeInt)

Fifth argument to `NewCLIFlag` is used to define what is the type of flag, is
it required etc. It's an integer value and the following `const`s are
available:

    * TypePathFile - flag is a path to a file (string);
    * MustExist - if added along with CLIFlagTypePathFile then path must exist;
    * Required - flag is required (this does not work with bool flag);
    * TypeString - flag is a string;
    * TypeBool - flag is boolean and will have a value of "true" or "false".

Check cli_flag.go for more information on flag types.

Finally, let's create functions to handle our commands. In below code, you can
see that method Flag on CLI instance (passed as first argument) can be
used to get a flag value.

    func InitHandler(c *cli.CLI) int {
        fmt.Fprintf(os.Stdout, "Template path: %s\n", c.Flag("template"))
        return 0
    }

    func StartHandler(c *cli.CLI) int {
        fmt.Fprintf(os.Stdout, "Username: %s\n", c.Flag("username"))
        return 0
    }

And in the end of main() func:

        os.Exit(myCLI.Run(os.Stdout, os.Stderr))

*/
package cli
