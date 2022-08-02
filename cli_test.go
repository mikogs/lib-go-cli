package cli

import (
	"os"
	"testing"
)

func h(c *CLI) int {
	return 0
}

func createCLI() *CLI {
	c := NewCLI("Example CLI", "Silly app", "Author <a@example.com>")

	cmd1 := c.AddCmd("command", "Prints out something", h)
	cmd1.AddFlag("bool", "b", "", "Boolean flag", TypeBool, nil)
	cmd1.AddFlag("input", "i", "filepath", "Path to a file", TypePathFile|Required, nil)
	cmd1.AddFlag("title", "t", "title", "Title of the project", TypeString|Required, nil)
	cmd1.AddFlag("desc", "d", "description", "Description of the project", TypeString, nil)

	cmd2 := c.AddCmd("anotherone", "Initialises project", h)
	cmd2.AddFlag("int", "i", "int", "Integer flag", TypeInt|Required, nil)
	cmd2.AddFlag("float", "f", "float", "Float flag", TypeFloat|Required, nil)
	cmd2.AddFlag("anum", "", "alphanumeric", "Alphanumeric flag", TypeAlphanumeric|Required, nil)

	cmd3 := c.AddCmd("three", "Third command that does something", h)
	cmd3.AddFlag("many-ints", "i", "int,int,...", "Many integers comma-delimetered", TypeInt|AllowMany, nil)
	cmd3.AddFlag("many-floats", "f", "float;float;...", "Many floats semicolon-delimetered", TypeFloat|AllowMany|ManySeparatorSemiColon|Required, nil)
	cmd3.AddFlag("many-anums", "a", "anum:anum:...", "Many alphanumeric colon-delimetered", TypeAlphanumeric|AllowMany|ManySeparatorColon, nil)
	cmd3.AddFlag("more", "m", "alphanumeric+dot+uscore", "Alphanumeric flag with additional dots and underscore allowed", TypeAlphanumeric|AllowDots|AllowUnderscore|Required, nil)

	cmd4 := c.AddCmd("play", "Play the game on a specific map", h)
	cmd4.AddFlag("level", "l", "1", "Starting level (1-50)", TypeInt|Required, nil)
	cmd4.AddFlag("difficulty", "d", "3", "Difficulty (1-5)", TypeInt, nil)
	cmd4.AddArg("map", "MAP", "Name of the map, eg. arena", TypeString|Required)
	cmd4.AddArg("opponents", "OPPONENTS", "Number of oponents, default 5", TypeInt|Required)
	cmd4.AddArg("foes", "FOES", "Number of foes, default 0", TypeInt)

	cmd5 := c.AddCmd("overwrite_arg", "Overwrite argument when flag is passed", h)
	cmd5.AddFlag("overwrite", "o", "", "Make argument not required", TypeBool, func(c *CLICmd) {
		f := c.GetArg("notrequired")
		f.nflags = TypeString
	})
	cmd5.AddArg("notrequired", "NOTREQUIRED", "Argument required when no -o", TypeString|Required)

	c.AddFlagToCmds("all", "x", "", "Flag added to all commands", TypeInt, nil)
	c.AddArgToCmds("all", "ALL", "Arg added to all commands", TypeString)

	return c
}

func assertExitCode(t *testing.T, cli *CLI, a []string, c int) {
	os.Args = a
	f, _ := os.Open("/dev/null")
	defer f.Close()
	got := cli.Run(f, f)
	want := c
	if got != want {
		t.Errorf("got %d want %d\n", got, want)
	}
}

func TestFlags(t *testing.T) {
	c := createCLI()

	t.Run("exit with code 0 when no flags or help", func(t *testing.T) {
		assertExitCode(t, c, []string{"test"}, 0)
		assertExitCode(t, c, []string{"test", "-h"}, 0)
		assertExitCode(t, c, []string{"test", "--help"}, 0)
		assertExitCode(t, c, []string{"test", "command", "-h"}, 0)
		assertExitCode(t, c, []string{"test", "command", "--help"}, 0)
	})

	t.Run("exit with code 1 when required flag is missing", func(t *testing.T) {
		assertExitCode(t, c, []string{"test", "command", "-b", "-t", "title", "-d", "desc"}, 1)
		assertExitCode(t, c, []string{"test", "command", "-i", "cli_test.go"}, 1)
		assertExitCode(t, c, []string{"test", "anotherone"}, 1)
		assertExitCode(t, c, []string{"test", "three", "-i", "1,2,3"}, 1)
	})

	t.Run("exit with code 1 when value is invalid", func(t *testing.T) {
		assertExitCode(t, c, []string{"test", "command", "-i", "nonexistingfile", "-t", "title"}, 1)
		assertExitCode(t, c, []string{"test", "anotherone", "--int", "aaaa", "--float", "123.12", "--anum", "validvalue"}, 1)
		assertExitCode(t, c, []string{"test", "anotherone", "--int", "123", "--float", "123", "--anum", "validvalue"}, 1)
		assertExitCode(t, c, []string{"test", "anotherone", "--int", "123", "--float", "123.12", "--anum", "^^4443####"}, 1)
		assertExitCode(t, c, []string{"test", "three", "-i", "aasd,asda", "-f", "12.33", "-a", "user1", "-m", "user.1"}, 1)
		assertExitCode(t, c, []string{"test", "three", "-i", "1,2,3", "-f", "12,33", "-a", "user1", "-m", "user.1"}, 1)
		assertExitCode(t, c, []string{"test", "three", "-i", "1,2,3", "-f", "23.23,24.24,25.25", "-a", "user1.", "-m", "user_1"}, 1)
	})

	t.Run("exit with code 1 when arg is missing", func(t *testing.T) {
		assertExitCode(t, c, []string{"test", "play", "--level", "1", "-d", "4"}, 1)
		assertExitCode(t, c, []string{"test", "play", "map", "4", "5", "--level", "1", "-d", "4"}, 1)
		assertExitCode(t, c, []string{"test", "play", "-l", "1", "-d", "4", "winter"}, 1)
	})

	t.Run("exit with code 1 when arg has invalid value", func(t *testing.T) {
		assertExitCode(t, c, []string{"test", "play", "-l", "1", "-d", "4", "winter", "five"}, 1)
	})

	t.Run("exit with code 0 when required arg is no longer required because it was overwritten by a flag", func(t *testing.T) {
		assertExitCode(t, c, []string{"test", "overwrite_arg", "REQUIRED_ARG_HERE"}, 0)
		assertExitCode(t, c, []string{"test", "overwrite_arg", "-o"}, 0)
	})
}
