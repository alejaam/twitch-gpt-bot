package twitch

import (
	"fmt"
	"strings"
)

type ParsedLine struct {
	Raw       string
	IsPing    bool
	PingParam string
	IsPrivmsg bool
	User      string
	Channel   string
	Text      string
}

// ParseLine parses a single IRC line received from Twitch chat server.
func ParseLine(line string) (*ParsedLine, error) {
	line = strings.TrimRight(line, "\r\n")
	res := &ParsedLine{Raw: line}

	if strings.HasPrefix(line, "PING") {
		res.IsPing = true
		parts := strings.SplitN(line, " ", 2)
		if len(parts) > 1 {
			res.PingParam = parts[1]
		}
		return res, nil
	}

	// Example: :username!username@username.tmi.twitch.tv PRIVMSG #channel :message
	if !strings.HasPrefix(line, ":") {
		return res, nil
	}

	prefixEnd := strings.IndexByte(line, ' ')
	if prefixEnd == -1 {
		return res, nil
	}

	prefix := line[1:prefixEnd]
	rest := strings.TrimLeft(line[prefixEnd+1:], " ")

	cmdEnd := strings.IndexByte(rest, ' ')
	if cmdEnd == -1 {
		return res, nil
	}

	command := rest[:cmdEnd]
	trailing := rest[cmdEnd+1:]

	if command != "PRIVMSG" {
		return res, nil
	}

	res.IsPrivmsg = true

	// Extract user from prefix: username!username@username.tmi.twitch.tv
	if userEnd := strings.IndexByte(prefix, '!'); userEnd != -1 {
		res.User = prefix[:userEnd]
	} else {
		res.User = prefix
	}

	// Parse channel and text from trailing: #channel :message text
	parts := strings.SplitN(trailing, " :", 2)
	res.Channel = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		res.Text = parts[1]
	}

	return res, nil
}

// FormatPrivmsg formats an outgoing PRIVMSG string for Twitch IRC.
func FormatPrivmsg(channel, text string) string {
	channel = NormalizeChannel(channel)
	return fmt.Sprintf("PRIVMSG %s :%s\r\n", channel, text)
}

// NormalizeChannel ensures a channel starts with '#'.
func NormalizeChannel(ch string) string {
	ch = strings.TrimSpace(ch)
	if !strings.HasPrefix(ch, "#") {
		return "#" + ch
	}
	return ch
}
