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

	"github.com/Dyastin-0/mrps/internal/metrics"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/Dyastin-0/mrps/pkg/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	sshUtil "golang.org/x/crypto/ssh"
)

type CommandMessage struct {
	SSHCommand string `json:"SSHCommand"`
	SessionID  string `json:"SessionID"`
}

type message struct {
	Type    string `json:"type"`
	Message string `json:"message"`
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

	modes := sshUtil.TerminalModes{
		sshUtil.ECHO:          1,
		sshUtil.TTY_OP_ISPEED: 14400,
		sshUtil.TTY_OP_OSPEED: 14400,
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to start ssh session: %w", err)
	}

	if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
		client.Close()
		cancel()
		return nil, fmt.Errorf("failed to request pty: %w", err)
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

	go streamOuput(stdoutPipe, wsID, "stdout")
	go streamOuput(stderrPipe, wsID, "stderr")

	go func() {
		recv, ok := ws.Clients.Listen(wsID)
		shortenWSid := "..." + wsID[max(0, len(wsID)-10):]
		if !ok {
			log.Error().Str("type", "connection").Str("status", "failed").Str("client", shortenWSid).Msg("ssh")
			return
		}

		metrics.ActiveSSHConns.Inc()

		newSessionID := uuid.New()
		session := message{
			Type:    "sshSessionID",
			Message: newSessionID,
		}
		sessionBytes, _ := json.Marshal(session)

		ws.Clients.Send(wsID, sessionBytes)

		for msg := range recv {
			var cmdMsg CommandMessage
			if err := json.Unmarshal(msg, &cmdMsg); err != nil {
				log.Error().Err(err).Msg("ssh")
				continue
			}

			if cmdMsg.SessionID != newSessionID {
				shortenSession := "..." + cmdMsg.SessionID[max(0, len(cmdMsg.SessionID)-10):]
				log.Error().Err(fmt.Errorf("mismatched session id")).Str("status", "closed").Str("session", shortenSession).Str("client", shortenWSid).Msg("ssh")
				break
			}

			if cmdMsg.SSHCommand == "\u0004" {
				notif := message{
					Type:    "notif",
					Message: "ssh disconnected",
				}
				notifByte, _ := json.Marshal(notif)

				ws.Clients.Send(wsID, notifByte)
				break
			}

			_, err := stdinPipe.Write([]byte(cmdMsg.SSHCommand))
			if err != nil {
				log.Error().Err(err).Msg("failed to write to stdin")
				break
			}
		}

		cancel()
	}()

	go func() {
		<-ctx.Done()
		log.Info().Str("status", "disconnected").Str("client", "..."+wsID[max(0, len(wsID)-10):]).Msg("ssh")
		session.Close()
		client.Close()
		metrics.ActiveSSHConns.Dec()
	}()

	return cancel, nil
}

func streamOuput(reader io.Reader, wsID, streamType string) {
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

			ws.Clients.Send(wsID, byteMessage)
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
