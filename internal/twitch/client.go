package twitch

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

const defaultTwitchIRCAddress = "irc.chat.twitch.tv:6697"

type ClientConfig struct {
	Username   string
	OAuthToken string
	Channel    string
	Address    string
}

type Client struct {
	username   string
	token      string
	channel    string
	address    string
	OnMessage  func(msg *ParsedLine)

	mu   sync.Mutex
	conn net.Conn
}

func NewClient(cfg ClientConfig) *Client {
	addr := strings.TrimSpace(cfg.Address)
	if addr == "" {
		addr = defaultTwitchIRCAddress
	}

	token := strings.TrimSpace(cfg.OAuthToken)
	if token != "" && !strings.HasPrefix(token, "oauth:") {
		token = "oauth:" + token
	}

	return &Client{
		username: strings.TrimSpace(cfg.Username),
		token:    token,
		channel:  NormalizeChannel(cfg.Channel),
		address:  addr,
	}
}

// Run connects to Twitch IRC with TLS and automatically reconnects if disconnected.
func (c *Client) Run(ctx context.Context) error {
	backoff := 1 * time.Second
	const maxBackoff = 30 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slog.Info("connecting to Twitch IRC", "address", c.address, "channel", c.channel)
		dialer := &tls.Dialer{
			NetDialer: &net.Dialer{Timeout: 10 * time.Second},
		}

		conn, err := dialer.DialContext(ctx, "tcp", c.address)
		if err != nil {
			slog.Error("failed to connect to Twitch IRC", "error", err)
			select {
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		backoff = 1 * time.Second
		slog.Info("connected to Twitch IRC")

		err = c.ServeConn(ctx, conn)
		if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) && ctx.Err() == nil {
			slog.Warn("Twitch IRC connection ended with error", "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

// ServeConn performs handshake and reads lines from an established connection.
func (c *Client) ServeConn(ctx context.Context, conn net.Conn) error {
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.mu.Unlock()
		_ = conn.Close()
	}()

	// Handshake
	if err := c.writeRaw(fmt.Sprintf("PASS %s\r\n", c.token)); err != nil {
		return fmt.Errorf("send PASS: %w", err)
	}
	if err := c.writeRaw(fmt.Sprintf("NICK %s\r\n", c.username)); err != nil {
		return fmt.Errorf("send NICK: %w", err)
	}
	if err := c.writeRaw(fmt.Sprintf("JOIN %s\r\n", c.channel)); err != nil {
		return fmt.Errorf("send JOIN: %w", err)
	}

	reader := bufio.NewReader(conn)

	errCh := make(chan error, 1)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				errCh <- err
				return
			}

			parsed, err := ParseLine(line)
			if err != nil {
				continue
			}

			if parsed.IsPing {
				pongMsg := "PONG :tmi.twitch.tv\r\n"
				if parsed.PingParam != "" {
					pongMsg = fmt.Sprintf("PONG %s\r\n", parsed.PingParam)
				}
				_ = c.writeRaw(pongMsg)
				continue
			}

			if parsed.IsPrivmsg && c.OnMessage != nil {
				c.OnMessage(parsed)
			}
		}
	}()

	select {
	case <-ctx.Done():
		_ = conn.Close()
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (c *Client) SendMessage(ctx context.Context, text string) error {
	line := FormatPrivmsg(c.channel, text)
	return c.writeRaw(line)
}

func (c *Client) writeRaw(line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	_, err := c.conn.Write([]byte(line))
	return err
}
