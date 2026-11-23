package twitch

import "testing"

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantUser    string
		wantChannel string
		wantText    string
		wantIsMsg   bool
		wantIsPing  bool
	}{
		{
			name:        "Standard PRIVMSG",
			raw:         ":coolviewer!coolviewer@coolviewer.tmi.twitch.tv PRIVMSG #streamer :!gpt tell me a joke",
			wantUser:    "coolviewer",
			wantChannel: "#streamer",
			wantText:    "!gpt tell me a joke",
			wantIsMsg:   true,
			wantIsPing:  false,
		},
		{
			name:        "PRIVMSG with colon in content",
			raw:         ":viewer!viewer@viewer.tmi.twitch.tv PRIVMSG #streamer :http://example.com is cool :D",
			wantUser:    "viewer",
			wantChannel: "#streamer",
			wantText:    "http://example.com is cool :D",
			wantIsMsg:   true,
			wantIsPing:  false,
		},
		{
			name:       "PING message",
			raw:        "PING :tmi.twitch.tv",
			wantIsPing: true,
			wantIsMsg:  false,
		},
		{
			name:       "Other IRC command (JOIN)",
			raw:        ":coolviewer!coolviewer@coolviewer.tmi.twitch.tv JOIN #streamer",
			wantIsMsg:  false,
			wantIsPing: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := ParseLine(tc.raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if msg.IsPing != tc.wantIsPing {
				t.Errorf("IsPing got %v, want %v", msg.IsPing, tc.wantIsPing)
			}
			if msg.IsPrivmsg != tc.wantIsMsg {
				t.Errorf("IsPrivmsg got %v, want %v", msg.IsPrivmsg, tc.wantIsMsg)
			}
			if tc.wantIsMsg {
				if msg.User != tc.wantUser {
					t.Errorf("User got %q, want %q", msg.User, tc.wantUser)
				}
				if msg.Channel != tc.wantChannel {
					t.Errorf("Channel got %q, want %q", msg.Channel, tc.wantChannel)
				}
				if msg.Text != tc.wantText {
					t.Errorf("Text got %q, want %q", msg.Text, tc.wantText)
				}
			}
		})
	}
}

func TestFormatPrivmsg(t *testing.T) {
	line := FormatPrivmsg("streamer", "@viewer hello world!")
	expected := "PRIVMSG #streamer :@viewer hello world!\r\n"
	if line != expected {
		t.Fatalf("expected %q, got %q", expected, line)
	}
}
