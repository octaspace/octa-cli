package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"

	"github.com/octaspace/octa/internal/vpnd"
	vpncore "github.com/octaspace/octa/vpnd"
	"github.com/spf13/cobra"
)

type daemon struct {
	mu     sync.Mutex
	active *vpncore.Tunnel
}

func (d *daemon) connect(config string) vpnd.Response {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.active != nil {
		d.active.Close()
		d.active = nil
	}

	t, err := vpncore.NewTunnel(config)
	if err != nil {
		return vpnd.Response{OK: false, Error: err.Error()}
	}

	d.active = t
	return vpnd.Response{OK: true, Interface: t.Interface()}
}

func (d *daemon) disconnect() vpnd.Response {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.active == nil {
		return vpnd.Response{OK: false, Error: "no active tunnel"}
	}

	d.active.Close()
	d.active = nil
	return vpnd.Response{OK: true}
}

func (d *daemon) status() vpnd.Response {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.active != nil {
		return vpnd.Response{OK: true, Interface: d.active.Interface()}
	}
	return vpnd.Response{OK: true}
}

func (d *daemon) handle(conn net.Conn) {
	defer conn.Close()

	var req vpnd.Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		json.NewEncoder(conn).Encode(vpnd.Response{OK: false, Error: "invalid request"})
		return
	}

	var resp vpnd.Response
	switch req.Cmd {
	case "connect":
		if req.Config == "" {
			resp = vpnd.Response{OK: false, Error: "config is required"}
		} else {
			resp = d.connect(req.Config)
		}
	case "disconnect":
		resp = d.disconnect()
	case "status":
		resp = d.status()
	default:
		resp = vpnd.Response{OK: false, Error: fmt.Sprintf("unknown command: %q", req.Cmd)}
	}

	json.NewEncoder(conn).Encode(resp)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the VPN daemon (requires root)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if runtime.GOOS != "windows" && os.Getuid() != 0 {
			fmt.Fprintln(os.Stderr, "octa daemon must be run as root")
			os.Exit(1)
		}

		sock := vpnd.SocketPath()

		if runtime.GOOS == "windows" {
			os.MkdirAll(`C:\ProgramData\octa-vpn`, 0700)
		}

		os.Remove(sock)

		l, err := net.Listen("unix", sock)
		if err != nil {
			return fmt.Errorf("listen %s: %w", sock, err)
		}
		defer l.Close()

		if runtime.GOOS != "windows" {
			if err := os.Chmod(sock, 0666); err != nil {
				return fmt.Errorf("chmod socket: %w", err)
			}
		}

		log.Printf("octa daemon listening on %s", sock)

		d := &daemon{}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sig
			log.Println("shutting down")
			d.mu.Lock()
			if d.active != nil {
				d.active.Close()
			}
			d.mu.Unlock()
			os.Remove(sock)
			os.Exit(0)
		}()

		for {
			conn, err := l.Accept()
			if err != nil {
				log.Printf("accept: %v", err)
				continue
			}
			go d.handle(conn)
		}
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
