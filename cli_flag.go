package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
)

const (
	// Required sets flag to be required.
	Required = 1
	// TypeString sets flag to be string.
	TypeString = 8
	// TypePathFile sets flag to be path to a file.
	TypePathFile = 16
	// TypeBool sets flag to be boolean.
	TypeBool = 32
	// TypeInt sets flag to be integer.
	TypeInt = 64
	// TypeFloat sets flag to be float.
	TypeFloat = 128
	// TypeAlphanumeric sets flag to be alphanumeric.
	TypeAlphanumeric = 256
	// MustExist sets flag must point to an existing file (required TypePathFile to be added as well).
	MustExist = 512
	// AllowMany allows flag to have more than one value separated by comma by default.
	// For example: AllowMany with TypeInt allows values like: 123 or 123,455,666 or 12,222
	// AllowMany works only with TypeInt, TypeFloat and TypeAlphanumeric.
	AllowMany = 1024
	// ManySeparatorColon works with AllowMany and sets colon to be the value separator, instead of colon.
	ManySeparatorColon = 2048
	// ManySeparatorSemiColon works with AllowMany and sets semi-colon to be the value separator.
	ManySeparatorSemiColon = 4096
	// AllowDots can be used only with TypeAlphanumeric and additionally allows flag to have dots.
	AllowDots = 8192
	// AllowUnderscore can be used only with TypeAlphanumeric and additionally allows flag to have underscore chars.
	AllowUnderscore = 16384
	// AllowHyphen can be used only with TypeAlphanumeric and additionally allows flag to have hyphen chars.
	AllowHyphen = 32768
	// TypeEmail sets flag to be an email (value is checked against a regular expression)
	TypeEmail = 65536
	// TypeFQDN sets flag to be a FQDN
	TypeFQDN = 131072
	// TypePathDir sets flag to be a directory
	TypePathDir = 262144
	// TypePathRegularFile sets flag to be a regular file
	TypePathRegularFile = 524288
	// ValidJSON sets flag to be a valid JSON. If it's a file then it's contents is checked. Otherwise it's the value
	ValidJSON = 1048576
)

// CLIFlag represends flag. It has a name, alias, description, value that is shown when printing help and configuration which is an integer value. It can be for example Required|TypePathFile|MustExist.
type CLIFlag struct {
	name      string
	alias     string
	helpValue string
	desc      string
	nflags    int32
	fn        func(*CLICmd)
}

// GetHelpLine returns flag usage info that is used when printing help.
func (c *CLIFlag) GetHelpLine() string {
	s := " "
	if c.alias == "" {
		s += " \t"
	} else {
		s += fmt.Sprintf(" -%s,\t", c.alias)
	}
	s += fmt.Sprintf(" --%s %s \t%s\n", c.name, c.helpValue, c.desc)
	return s
}

// IsRequireValue returns true when flag requires a value (only bool one returns false).
func (c *CLIFlag) IsRequireValue() bool {
	return c.nflags&TypeString > 0 || c.nflags&TypePathFile > 0 || c.nflags&TypePathRegularFile > 0 || c.nflags&TypePathDir > 0 || c.nflags&TypeInt > 0 || c.nflags&TypeFloat > 0 || c.nflags&TypeAlphanumeric > 0
}

// ValidateValue takes value coming from --NAME and -ALIAS and validates it.
func (c *CLIFlag) ValidateValue(isArg bool, nz string, az string) error {
	// both alias and name cannot be set
	if nz != "" && az != "" {
		return errors.New(fmt.Sprintf("Both -%s and --%s passed", c.alias, c.name))
	}

	label := "Flag"
	if isArg {
		label = "Argument"
	}

	nlabel := c.name
	if isArg {
		nlabel = c.helpValue
	}

	// empty
	if (c.nflags&Required > 0) && (nz == "" && az == "") {
		if c.nflags&TypeString > 0 || c.nflags&TypePathFile > 0 || c.nflags&TypePathRegularFile > 0 || c.nflags&TypePathDir > 0 || c.nflags&TypeInt > 0 || c.nflags&TypeFloat > 0 || c.nflags&TypeAlphanumeric > 0 {
			return errors.New(fmt.Sprintf("%s %s is missing", label, nlabel))
		}
	}
	// string does not need any additional checks apart from the above one
	if c.nflags&TypeString > 0 {
		return nil
	}
	v := az
	if nz != "" {
		v = nz
	}

	if c.nflags&Required > 0 || v != "" {
		// if flag is a file and have to exist
		if c.nflags&TypePathFile > 0 {
			if _, err := os.Stat(v); os.IsNotExist(err) {
				return errors.New("File " + v + " from " + nlabel + " does not exist")
			}
			return nil
		}
		// if flag is a regular file and have to exist
		if c.nflags&TypePathRegularFile > 0 {
			fileInfo, err := os.Stat(v)
			if os.IsNotExist(err) {
				return errors.New("File " + v + " from " + nlabel + " does not exist")
			}
			if !fileInfo.Mode().IsRegular() {
				return errors.New("Path " + v + " from " + nlabel + " is not a regular file")
			}
			if c.nflags&ValidJSON > 0 {
				dat, err := os.ReadFile(v)
				if err != nil {
					return errors.New(v + " " + nlabel + " cannot be opened")
				}
				if !json.Valid(dat) {
					return errors.New(v + " " + nlabel + " is not a valid JSON")
				}
			}
			return nil
		}
		// if flag is a directory and have to exist
		if c.nflags&TypePathDir > 0 {
			fileInfo, err := os.Stat(v)
			if os.IsNotExist(err) {
				return errors.New("Directory " + v + " from " + nlabel + " does not exist")
			}
			if !fileInfo.IsDir() {
				return errors.New("Path " + v + " from " + nlabel + " is not a directory")
			}
			return nil
		}
		// int, float, alphanumeric - single or many, separated by various chars
		var reType string
		var reValue string
		// set regexp part just for the type (eg. int, float, anum)
		if c.nflags&TypeInt > 0 {
			reType = "[0-9]+"
		} else if c.nflags&TypeFloat > 0 {
			reType = "[0-9]{1,16}\\.[0-9]{1,16}"
		} else if c.nflags&TypeAlphanumeric > 0 {
			// alphanumeric + additional characters
			if c.nflags&AllowHyphen > 0 && c.nflags&AllowUnderscore > 0 && c.nflags&AllowDots > 0 {
				reType = "[0-9a-zA-Z_\\.\\-]+"
			} else if c.nflags&AllowUnderscore > 0 && c.nflags&AllowDots > 0 {
				reType = "[0-9a-zA-Z_\\.]+"
			} else if c.nflags&AllowUnderscore > 0 && c.nflags&AllowHyphen > 0 {
				reType = "[0-9a-zA-Z_\\-]+"
			} else if c.nflags&AllowDots > 0 && c.nflags&AllowHyphen > 0 {
				reType = "[0-9a-zA-Z\\.\\-]+"
			} else if c.nflags&AllowUnderscore > 0 {
				reType = "[0-9a-zA-Z_]+"
			} else if c.nflags&AllowDots > 0 {
				reType = "[0-9a-zA-Z\\.]+"
			} else {
				reType = "[0-9a-zA-Z]+"
			}
		}
		// create the final regexp depending on if single or many values are allowed
		if c.nflags&AllowMany > 0 {
			var d string
			if c.nflags&ManySeparatorColon > 0 {
				d = ":"
			} else if c.nflags&ManySeparatorSemiColon > 0 {
				d = ";"
			} else {
				d = ","
			}
			reValue = "^" + reType + "(" + d + reType + ")*$"
		} else {
			reValue = "^" + reType + "$"
		}
		m, err := regexp.MatchString(reValue, v)
		if err != nil || !m {
			return errors.New(label + " " + nlabel + " has invalid value")
		}
	}
	return nil
}

// NewCLIFlag creates instance of CLIFlag and returns it.
func NewCLIFlag(n string, a string, hv string, d string, nf int32, fn func(*CLICmd)) *CLIFlag {
	f := &CLIFlag{name: n, alias: a, helpValue: hv, desc: d, nflags: nf, fn: fn}
	return f
}
