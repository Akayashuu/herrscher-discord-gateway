package discord

import (
	"context"
	"fmt"
	"time"

	"github.com/Akayashuu/dctl"
	"github.com/Akayashuu/herrscher-contracts"
)

// Discord channel-type ints (GUILD_CATEGORY is 4; dctl exports ChannelForum=15).
const channelCategory = 4

// ChannelAdmin adapts the dctl client to serve.ChannelAdmin: session channel
// creation/archival and posting.
type ChannelAdmin struct{ c *dctl.Client }

func NewChannelAdmin(c *dctl.Client) *ChannelAdmin { return &ChannelAdmin{c: c} }

func (a *ChannelAdmin) Kind(ctx context.Context, id string) (string, error) {
	t, err := a.c.ChannelType(ctx, id)
	if err != nil {
		return "", err
	}
	switch t {
	case channelCategory:
		return "category", nil
	case dctl.ChannelForum:
		return "forum", nil
	default:
		return "", nil
	}
}

func (a *ChannelAdmin) CreateUnder(ctx context.Context, parentID, name string) (string, error) {
	ch, err := a.c.CreateChannelUnder(ctx, parentID, name)
	if err != nil {
		return "", err
	}
	if ch == nil {
		return "", nil
	}
	return ch.ID, nil
}

func (a *ChannelAdmin) ForumPost(ctx context.Context, forumID, name, content string) (string, error) {
	ch, err := a.c.ForumPost(ctx, forumID, name, content)
	if err != nil {
		return "", err
	}
	if ch == nil {
		return "", nil
	}
	return ch.ID, nil
}

func (a *ChannelAdmin) Archive(ctx context.Context, id string) error {
	return a.c.ArchiveChannel(ctx, id)
}

func (a *ChannelAdmin) Send(ctx context.Context, channelID, content string) error {
	_, err := a.c.Send(ctx, channelID, content)
	return err
}

// Platform adapts the dctl client to contracts.Platform (the bridge's read/
// channel-bootstrap/reaction/status/select-menu surface).
type Platform struct{ c *dctl.Client }

func NewPlatform(c *dctl.Client) *Platform { return &Platform{c: c} }

func (p *Platform) Enabled() bool          { return p.c.Enabled() }
func (p *Platform) DefaultChannel() string { return p.c.DefaultChannel() }

func (p *Platform) EnsureChannel(ctx context.Context, parentID, name string) (contracts.Channel, error) {
	ch, err := p.c.EnsureChannel(ctx, parentID, name)
	if err != nil {
		return contracts.Channel{}, err
	}
	if ch == nil {
		return contracts.Channel{}, nil
	}
	return contracts.Channel{ID: ch.ID, Name: ch.Name}, nil
}

func (p *Platform) Read(ctx context.Context, channelID string, limit int, after string) ([]contracts.Message, error) {
	msgs, err := p.c.Read(ctx, channelID, limit, after)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.Message, 0, len(msgs))
	for _, m := range msgs {
		atts := make([]contracts.Attachment, 0, len(m.Attachments))
		for _, a := range m.Attachments {
			atts = append(atts, contracts.Attachment{
				Filename:    a.Filename,
				URL:         a.URL,
				ContentType: a.ContentType,
				Size:        a.Size,
			})
		}
		out = append(out, contracts.Message{
			ID:          m.ID,
			ChannelID:   m.ChannelID,
			Content:     m.Content,
			AuthorID:    m.Author.ID,
			AuthorName:  m.Author.Username,
			AuthorBot:   m.Author.Bot,
			Attachments: atts,
		})
	}
	return out, nil
}

func (p *Platform) Unreact(ctx context.Context, channelID, messageID, emoji string) error {
	return p.c.Unreact(ctx, channelID, messageID, emoji)
}

func (p *Platform) UpsertStatusMessage(ctx context.Context, channelID, messageID, content string) (string, error) {
	return p.c.UpsertStatusMessage(ctx, channelID, messageID, content)
}

func (p *Platform) SendSelectMenu(ctx context.Context, channelID, replyTo, content, session string, opts []contracts.Choice) (contracts.MessageID, error) {
	out := make([]dctl.SelectOption, 0, len(opts))
	for _, o := range opts {
		out = append(out, dctl.SelectOption{Label: o.Label, Value: o.Value})
	}
	m, err := p.c.SendSelectMenu(ctx, channelID, replyTo, content, ChoiceCustomID(session), out)
	if err != nil {
		return "", err
	}
	if m == nil {
		return "", nil
	}
	return contracts.MessageID(m.ID), nil
}

// interactionToken is the opaque ResponseToken the Discord adapter packs: the
// interaction id and its short-lived token, needed to answer or edit it.
type interactionToken struct{ id, token string }

// Responder adapts the dctl client to contracts.CommandResponder. appID is the
// bot's application id, needed to edit a deferred interaction response.
type Responder struct {
	c     *dctl.Client
	appID string
}

func NewResponder(c *dctl.Client, appID string) *Responder { return &Responder{c: c, appID: appID} }

// token recovers the packed interaction id+token, rejecting a foreign token
// instead of panicking the host dispatch loop.
func token(tok contracts.ResponseToken) (interactionToken, error) {
	it, ok := tok.(interactionToken)
	if !ok {
		return interactionToken{}, fmt.Errorf("discord: invalid response token %T", tok)
	}
	return it, nil
}

func (r *Responder) Defer(ctx context.Context, tok contracts.ResponseToken, private bool) error {
	it, err := token(tok)
	if err != nil {
		return err
	}
	return r.c.DeferInteraction(ctx, it.id, it.token, private)
}

func (r *Responder) Respond(ctx context.Context, tok contracts.ResponseToken, resp contracts.CommandResponse) error {
	it, err := token(tok)
	if err != nil {
		return err
	}
	return r.c.RespondInteraction(ctx, it.id, it.token, dctl.Response{Content: resp.Content, Ephemeral: resp.Private})
}

func (r *Responder) Edit(ctx context.Context, tok contracts.ResponseToken, resp contracts.CommandResponse) error {
	it, err := token(tok)
	if err != nil {
		return err
	}
	return r.c.EditInteractionResponse(ctx, r.appID, it.token, dctl.Response{Content: resp.Content, Ephemeral: resp.Private})
}

func (r *Responder) Autocomplete(ctx context.Context, tok contracts.ResponseToken, choices []contracts.AutocompleteChoice) error {
	it, err := token(tok)
	if err != nil {
		return err
	}
	out := make([]dctl.AutocompleteChoice, 0, len(choices))
	for _, ch := range choices {
		out = append(out, dctl.AutocompleteChoice{Name: ch.Label, Value: ch.Value})
	}
	return r.c.RespondAutocomplete(ctx, it.id, it.token, out)
}

func (r *Responder) AckComponent(ctx context.Context, tok contracts.ResponseToken, content string) error {
	it, err := token(tok)
	if err != nil {
		return err
	}
	return r.c.AckComponent(ctx, it.id, it.token, content)
}

// Registrar adapts the slash-command registration to contracts.CommandRegistrar.
type Registrar struct{ c *dctl.Client }

func NewRegistrar(c *dctl.Client) *Registrar { return &Registrar{c: c} }

func (r *Registrar) Register(ctx context.Context) error { return RegisterCommands(ctx, r.c) }

// Prober adapts a cheap REST round-trip (/users/@me) to contracts.Prober.
type Prober struct{ c *dctl.Client }

func NewProber(c *dctl.Client) *Prober { return &Prober{c: c} }

func (p *Prober) Probe(ctx context.Context) (int64, error) {
	start := time.Now()
	_, err := p.c.AppID(ctx)
	return time.Since(start).Milliseconds(), err
}

// StatusReporter adapts the self-updating status message to contracts.StatusReporter.
type StatusReporter struct {
	c         *dctl.Client
	channelID string
}

func NewStatusReporter(c *dctl.Client, channelID string) *StatusReporter {
	return &StatusReporter{c: c, channelID: channelID}
}

func (s *StatusReporter) Upsert(ctx context.Context, prevID, content string) (string, error) {
	return s.c.UpsertStatusMessage(ctx, s.channelID, prevID, content)
}
