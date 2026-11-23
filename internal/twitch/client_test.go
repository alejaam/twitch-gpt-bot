package twitch

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClientHandshakeAndPing(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	serverReader := bufio.NewReader(serverConn)
	serverReceived := make(chan string, 10)

	go func() {
		for {
			line, err := serverReader.ReadString('\n')
			if err != nil {
				return
			}
			serverReceived <- strings.TrimRight(line, "\r\n")
		}
	}()

	client := NewClient(ClientConfig{
		Username:   "botuser",
		OAuthToken: "oauth:123456",
		Channel:    "#testchannel",
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use our pipe connection
	msgReceived := make(chan *ParsedLine, 1)
	client.OnMessage = func(msg *ParsedLine) {
		msgReceived <- msg
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.ServeConn(ctx, clientConn)
	}()

	// Verify handshake commands sent by client
	expectedCommands := []string{
		"PASS oauth:123456",
		"NICK botuser",
		"JOIN #testchannel",
	}

	for _, expected := range expectedCommands {
		select {
		case cmd := <-serverReceived:
			if cmd != expected {
				t.Fatalf("expected handshake cmd %q, got %q", expected, cmd)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timed out waiting for handshake %q", expected)
		}
	}

	// Server sends PING :tmi.twitch.tv
	_, _ = serverConn.Write([]byte("PING :tmi.twitch.tv\r\n"))
	select {
	case pong := <-serverReceived:
		if pong != "PONG :tmi.twitch.tv" {
			t.Fatalf("expected PONG response, got %q", pong)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for PONG")
	}

	// Server sends a chat message
	_, _ = serverConn.Write([]byte(":user1!user1@user1.tmi.twitch.tv PRIVMSG #testchannel :hello\r\n"))
	select {
	case msg := <-msgReceived:
		if msg.User != "user1" || msg.Text != "hello" {
			t.Fatalf("unexpected message parsed: %+v", msg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for parsed chat message")
	}

	// Client sends a message back
	err := client.SendMessage(ctx, "hey there!")
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}

	select {
	case outMsg := <-serverReceived:
		if outMsg != "PRIVMSG #testchannel :hey there!" {
			t.Fatalf("expected PRIVMSG sent, got %q", outMsg)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for client PRIVMSG")
	}

	cancel()
}
