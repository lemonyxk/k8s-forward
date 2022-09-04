/**
* @program: k8s-forward
*
* @description:
*
* @author: lemo
*
* @create: 2022-07-26 22:20
**/

package ssh

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/lemoyxk/console"
	"github.com/olekukonko/ts"
	"golang.org/x/crypto/ssh"
)

var client *ssh.Client
var withTTY bool

type Cmd struct {
	name    string
	args    []string
	cmd     *exec.Cmd
	session *ssh.Session

	Path string
	Args []string
	Env  []string
	Dir  string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	ExtraFiles   []*os.File
	SysProcAttr  *syscall.SysProcAttr
	Process      *os.Process
	ProcessState *os.ProcessState
}

func (c *Cmd) initCmd(cmd *exec.Cmd) *exec.Cmd {
	if c.Path != "" {
		cmd.Path = c.Path
	}
	if len(c.Args) > 0 {
		cmd.Args = c.Args
	}
	if len(c.Env) > 0 {
		cmd.Env = c.Env
	}
	if c.Dir != "" {
		cmd.Dir = c.Dir

	}
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	cmd.ExtraFiles = c.ExtraFiles
	cmd.SysProcAttr = c.SysProcAttr
	cmd.Process = c.Process
	cmd.ProcessState = c.ProcessState

	c.cmd = cmd

	return cmd
}

func (c *Cmd) initSession(session *ssh.Session) *ssh.Session {
	session.Stderr = c.Stderr
	session.Stdin = c.Stdin
	session.Stdout = c.Stdout

	if withTTY {
		setTTY(session)
	}

	c.session = session

	return session
}

func (c *Cmd) Run() error {
	if client == nil {
		var cmd = exec.Command(c.name, c.args...)
		c.initCmd(cmd)
		return cmd.Run()
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}

	defer func() { _ = session.Close() }()

	return c.initSession(session).Run(c.name + " " + strings.Join(c.args, " "))
}

func (c *Cmd) Start() error {
	if client == nil {
		var cmd = exec.Command(c.name, c.args...)
		return c.initCmd(cmd).Start()
	}

	session, err := client.NewSession()
	if err != nil {
		return err
	}

	return c.initSession(session).Start(c.name + " " + strings.Join(c.args, " "))
}

func (c *Cmd) Wait() error {
	if client == nil {
		return c.cmd.Wait()
	}

	defer func() { _ = c.session.Close() }()

	return c.session.Wait()
}

func (c *Cmd) Output() ([]byte, error) {
	if client == nil {
		var cmd = exec.Command(c.name, c.args...)
		return c.initCmd(cmd).Output()
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	defer func() { _ = session.Close() }()

	return c.initSession(session).Output(c.name + " " + strings.Join(c.args, " "))
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	if client == nil {
		var cmd = exec.Command(c.name, c.args...)
		return c.initCmd(cmd).CombinedOutput()
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}

	defer func() { _ = c.session.Close() }()

	return c.initSession(session).CombinedOutput(c.name + " " + strings.Join(c.args, " "))
}

func (c *Cmd) StdinPipe() (io.WriteCloser, error) {
	if client == nil {
		return c.cmd.StdinPipe()
	}

	return c.session.StdinPipe()
}

func (c *Cmd) StdoutPipe() (io.Reader, error) {
	if client == nil {
		return c.cmd.StdoutPipe()
	}

	return c.session.StdoutPipe()
}

func (c *Cmd) StderrPipe() (io.Reader, error) {
	if client == nil {
		return c.cmd.StderrPipe()
	}

	return c.session.StderrPipe()
}

func InitSSHCommand(user, pass string, host string, port int) {
	c, err := Server(user, pass, host, port)
	if err != nil {
		panic(err)
	}

	client = c
}

func WithTTY() {
	withTTY = true
}

func setTTY(session *ssh.Session) {
	size, err := ts.GetSize()
	if err != nil {
		console.Error(err)
	}

	termWidth, termHeight := size.Col(), size.Row()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		console.Error(err)
	}
}

func Command(name string, args ...string) *Cmd {
	return &Cmd{name: name, args: args}
}
