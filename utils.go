package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

const (
	name  = "SKM"
	usage = "Manage your multiple SSH keys easily"

	checkSymbol = "\u2714 "
	crossSymbol = "\u2716 "

	publicKey  = "id_rsa.pub"
	privateKey = "id_rsa"
	defaultKey = "default"
)

var (
	storePath = filepath.Join(os.Getenv("HOME"), ".skm")
	sshPath   = filepath.Join(os.Getenv("HOME"), ".ssh")
)

func parseArgs() {
	if len(os.Args) == 1 {
		displayLogo()
	} else if len(os.Args) == 2 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "h" || os.Args[1] == "help" {
			displayLogo()
		}
	}
}

func execute(workDir, script string, args ...string) bool {
	cmd := exec.Command(script, args...)

	if workDir != "" {
		cmd.Dir = workDir
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		color.Red("%s%s", crossSymbol, err.Error())
		return false
	}

	return true
}

func clearKey() {
	//Remove private key if exists
	privateKeyPath := filepath.Join(sshPath, privateKey)
	if _, err := os.Stat(privateKeyPath); !os.IsNotExist(err) {
		os.Remove(privateKeyPath)
	}

	//Remove public key if exists
	publicKeyPath := filepath.Join(sshPath, publicKey)
	if _, err := os.Stat(publicKeyPath); !os.IsNotExist(err) {
		os.Remove(publicKeyPath)
	}
}

func deleteKey(alias string, key *SSHKey) {
	inUse := key.PrivateKey == parsePath(filepath.Join(sshPath, privateKey))

	if inUse {
		fmt.Print(color.BlueString("SSH key [%s] is currently in use, please confirm to delete it [y/n]: ", alias))
	} else {
		fmt.Print(color.BlueString("Please confirm to delete SSH key [%s] [y/n]: ", alias))
	}
	var input string
	fmt.Scan(&input)

	if input == "y" {

		if inUse {
			clearKey()
		}

		//Remove specified key by alias name
		if err := os.RemoveAll(filepath.Join(storePath, alias)); err == nil {
			color.Green("%sSSH key [%s] deleted!", checkSymbol, alias)
		} else {
			color.Red("%sFailed to delete SSH key [%s]!", crossSymbol, alias)
		}
	}
}

func createLink(alias string) {
	clearKey()

	//Create symlink for private key
	os.Symlink(filepath.Join(storePath, alias, privateKey), filepath.Join(sshPath, privateKey))
	//Create symlink for public key
	os.Symlink(filepath.Join(storePath, alias, publicKey), filepath.Join(sshPath, publicKey))
}

func loadSingleKey(keyPath string) *SSHKey {
	key := &SSHKey{}

	//Walkthrough SSH key store and load all the keys
	err := filepath.Walk(keyPath, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		if path == keyPath {
			return nil
		}

		if f.IsDir() {
			return nil
		}

		if strings.Contains(f.Name(), ".pub") {
			key.PublicKey = path
			return nil
		}

		//Check if key is in use
		key.PrivateKey = path

		if path == parsePath(filepath.Join(sshPath, privateKey)) {
			key.IsDefault = true
		}

		return nil
	})

	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
		return nil
	}

	if key.PublicKey != "" && key.PrivateKey != "" {
		return key
	}

	return nil
}

func parsePath(path string) string {
	fileInfo, err := os.Lstat(path)

	if err != nil {
		return ""
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		originFile, err := os.Readlink(path)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		return originFile
	}
	return path
}

func loadSSHKeys() map[string]*SSHKey {
	keys := map[string]*SSHKey{}

	//Walkthrough SSH key store and load all the keys
	err := filepath.Walk(storePath, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}

		if path == storePath {
			return nil
		}

		if f.IsDir() {
			//Load private/public keys
			key := loadSingleKey(path)

			if key != nil {
				keys[f.Name()] = key
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
	}

	return keys
}

func getBakFileName() string {
	return fmt.Sprintf("skm-%s.tar.gz", time.Now().Format("20060102150405"))
}
