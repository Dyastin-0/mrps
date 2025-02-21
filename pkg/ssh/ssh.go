package ssh

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	sshUtil "golang.org/x/crypto/ssh"
)

type CommandMessage struct {
	SSHCommand string `json:"SSHCommand"`
}

func StartSession(privateKey, instanceIP, hostKey, user, wsID string, wsConn *websocket.Conn) (context.CancelFunc, error) {
	if privateKey == "" || instanceIP == "" || user == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	ctx, cancel := context.WithCancel(context.Background())
	privateKey = strings.ReplaceAll(privateKey, `\n`, "\n")

	signer, err := sshUtil.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return cancel, fmt.Errorf("failed to parse private key: %w", err)
	}

	config := &sshUtil.ClientConfig{
		User: user,
		Auth: []sshUtil.AuthMethod{
			sshUtil.PublicKeys(signer),
		},
		HostKeyCallback: verifyHostKey(hostKey),
	}

	client, err := sshUtil.Dial("tcp", instanceIP+":22", config)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to ssh server: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to start ssh session: %w", err)
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	go streamOuput(stdoutPipe, wsConn, "stdout")
	go streamOuput(stderrPipe, wsConn, "stderr")

	go func() {
		recv, ok := ws.Clients.Listen(wsID)
		if !ok {
			log.Error().Str("status", "failed").Str("client", wsID).Msg("ssh")
			return
		}

		for msg := range recv {
			var cmdMsg CommandMessage
			if err := json.Unmarshal(msg, &cmdMsg); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal ssh command")
				continue
			}

			log.Info().Str("command", cmdMsg.SSHCommand).Msg("ssh")

			_, err := stdinPipe.Write([]byte(cmdMsg.SSHCommand + "\n"))
			if err != nil {
				log.Error().Err(err).Msg("failed to write to ssh stdin")
				break
			}
		}

		log.Info().Msg("client disconnected, stopping ssh command listener")
		cancel()
	}()

	go func() {
		<-ctx.Done()
		log.Info().Str("status", "disconnected").Msg("ssh")
		session.Close()
		client.Close()
	}()

	return cancel, nil
}

func streamOuput(reader io.Reader, wsConn *websocket.Conn, streamType string) {
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			message := struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    streamType,
				Message: string(buf[:n]),
			}

			byteMessage, err := json.Marshal(message)
			if err != nil {
				fmt.Println("failed to marshal message", err)
			}

			if err := wsConn.WriteMessage(websocket.TextMessage, byteMessage); err != nil {
				fmt.Println("Failed to send message to WebSocket:", err)
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Error reading from %s: %v\n", streamType, err)
			}
			break
		}
	}
}

func verifyHostKey(hostKey string) func(hostname string, remote net.Addr, key sshUtil.PublicKey) error {
	decodedHostKey, err := base64.StdEncoding.DecodeString(hostKey)
	if err != nil {
		return func(hostname string, remote net.Addr, key sshUtil.PublicKey) error {
			return fmt.Errorf("failed to decode host key: %w", err)
		}
	}

	return func(hostname string, remote net.Addr, key sshUtil.PublicKey) error {
		if bytes.Equal(decodedHostKey, key.Marshal()) {
			return nil
		}
		return fmt.Errorf("host key mismatch: expected %s, got %s", hostKey, base64.StdEncoding.EncodeToString(key.Marshal()))
	}
}
