package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/myopenfactory/edi-connector/v2/credentials"
	"golang.org/x/sys/windows"
)

func credentialsCommand(args []string) error {
	err := runCredentialsCommand(args)
	if errors.Is(err, os.ErrPermission) {
		return fmt.Errorf("%w\n(run this command in a console started as administrator)", err)
	}
	return err
}

func runCredentialsCommand(args []string) error {
	if len(args) == 0 {
		printCredentialsUsage()
		return fmt.Errorf("missing subcommand")
	}
	switch args[0] {
	case "list":
		return credentialsListCommand()
	case "set":
		return credentialsSetCommand(args[1:])
	case "set-b64":
		return credentialsSetEncodedCommand(args[1:])
	case "delete":
		return credentialsDeleteCommand(args[1:])
	case "migrate":
		return credentialsMigrateCommand()
	case "help":
		printCredentialsUsage()
		return nil
	default:
		printCredentialsUsage()
		return fmt.Errorf("unknown credentials subcommand: %s", args[0])
	}
}

func printCredentialsUsage() {
	fmt.Println(`usage: edi-connector credentials <command>

commands (administrator rights required):
  list                              list all stored credentials
  set <name> <username> [password]  create or update a credential, the
                                    password is prompted for if not given
  set-b64 <name> <user> <password>  like set, with base64 encoded username
                                    and password (used by the installer)
  delete <name>                     delete a credential
  migrate                           copy credentials of the current user from
                                    the Windows credential manager to the
                                    machine wide store (run as the former
                                    service user, existing ones are kept)`)
}

func credentialsListCommand() error {
	creds, err := credentials.ListCredentials()
	if err != nil {
		return err
	}
	if len(creds) == 0 {
		fmt.Println("no credentials stored")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tUSERNAME")
	for _, c := range creds {
		fmt.Fprintf(w, "%s\t%s\n", c.Name, c.Username)
	}
	return w.Flush()
}

func credentialsSetCommand(args []string) error {
	if len(args) < 2 {
		printCredentialsUsage()
		return fmt.Errorf("usage: edi-connector credentials set <name> <username> [password]")
	}
	name, username := args[0], args[1]
	var password string
	if len(args) >= 3 {
		password = args[2]
	} else {
		fmt.Print("Password: ")
		var err error
		password, err = readPassword()
		if err != nil {
			return err
		}
		fmt.Println()
	}
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}
	if err := credentials.SetCredential(name, username, password); err != nil {
		return err
	}
	fmt.Printf("credential %q updated\n", name)
	return nil
}

// credentialsSetEncodedCommand is used by the installer, the username and
// password are base64 encoded to survive the command line unchanged.
func credentialsSetEncodedCommand(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("usage: edi-connector credentials set-b64 <name> <base64 username> <base64 password>")
	}
	username, err := base64.StdEncoding.DecodeString(args[1])
	if err != nil {
		return fmt.Errorf("invalid username encoding: %w", err)
	}
	password, err := base64.StdEncoding.DecodeString(args[2])
	if err != nil {
		return fmt.Errorf("invalid password encoding: %w", err)
	}
	if len(password) == 0 {
		return fmt.Errorf("password must not be empty")
	}
	if err := credentials.SetCredential(args[0], string(username), string(password)); err != nil {
		return err
	}
	fmt.Printf("credential %q updated\n", args[0])
	return nil
}

func credentialsMigrateCommand() error {
	migrated, skipped, err := credentials.MigrateLegacyCredentials()
	for _, n := range migrated {
		fmt.Printf("migrated credential %q\n", n)
	}
	for _, n := range skipped {
		fmt.Printf("skipped credential %q, already exists\n", n)
	}
	if err != nil {
		return err
	}
	if len(migrated)+len(skipped) == 0 {
		fmt.Println("no legacy credentials found for the current user")
	}
	return nil
}

func credentialsDeleteCommand(args []string) error {
	if len(args) != 1 {
		printCredentialsUsage()
		return fmt.Errorf("usage: edi-connector credentials delete <name>")
	}
	if err := credentials.DeleteCredential(args[0]); err != nil {
		return err
	}
	fmt.Printf("credential %q deleted\n", args[0])
	return nil
}

func readPassword() (string, error) {
	stdin := windows.Handle(os.Stdin.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(stdin, &mode); err == nil {
		if err := windows.SetConsoleMode(stdin, mode&^windows.ENABLE_ECHO_INPUT); err == nil {
			defer windows.SetConsoleMode(stdin, mode)
		}
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
