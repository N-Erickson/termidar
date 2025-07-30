package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

func main() {
	port := "22"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	s, err := wish.NewServer(
		wish.WithAddress(":"+port),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(func(next ssh.Handler) ssh.Handler {
			return func(sess ssh.Session) {
				// Get terminal size
				pty, _, _ := sess.Pty()
				
				// Create command to run your existing main.go
				cmd := exec.Command("./termidar")
				
				// Set terminal size
				cmd.Env = append(os.Environ(),
					fmt.Sprintf("LINES=%d", pty.Window.Height),
					fmt.Sprintf("COLUMNS=%d", pty.Window.Width),
					"TERM=xterm-256color",
				)
				
				// Connect SSH session to command
				cmd.Stdin = sess
				cmd.Stdout = sess
				cmd.Stderr = sess
				
				// Run it
				if err := cmd.Run(); err != nil {
					fmt.Fprintf(sess, "Error: %v\n", err)
				}
				
				sess.Exit(0)
			}
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting SSH server on :%s", port)
	log.Fatal(s.ListenAndServe())
}