package types

// SlackMessage represents a Slack webhook message
// Reference: https://api.slack.com/messaging/webhooks
type SlackMessage struct {
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Blocks      []SlackBlock `json:"blocks,omitempty"`
	ThreadTS    string       `json:"thread_ts,omitempty"`
	UnfurlLinks bool         `json:"unfurl_links,omitempty"`
}

// SlackBlock represents a Slack Block Kit element
type SlackBlock struct {
	Type     string            `json:"type"`
	Text     *SlackTextObject  `json:"text,omitempty"`
	Fields   []SlackTextObject `json:"fields,omitempty"`
	Elements []SlackTextObject `json:"elements,omitempty"`
}

// SlackTextObject represents text within a Slack block
type SlackTextObject struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackResponse represents the response from Slack API
type SlackResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	TS      string `json:"ts,omitempty"`
	Channel string `json:"channel,omitempty"`
}
