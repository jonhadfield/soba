package slack

import (
	"encoding/json"
)

// RichTextBlock defines a new block of type rich_text.
// More Information: https://api.slack.com/changelog/2019-09-what-they-see-is-what-you-get-and-more-and-less
type RichTextBlock struct {
	Type     MessageBlockType  `json:"type"`
	BlockID  string            `json:"block_id,omitempty"`
	Elements []RichTextElement `json:"elements"`
}

func (b RichTextBlock) BlockType() MessageBlockType {
	return b.Type
}

// ID returns the ID of the block
func (s RichTextBlock) ID() string {
	return s.BlockID
}

func (e *RichTextBlock) UnmarshalJSON(b []byte) error {
	var raw struct {
		Type        MessageBlockType  `json:"type"`
		BlockID     string            `json:"block_id"`
		RawElements []json.RawMessage `json:"elements"`
	}
	if string(b) == "{}" {
		return nil
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	elems := make([]RichTextElement, 0, len(raw.RawElements))
	for _, r := range raw.RawElements {
		var s struct {
			Type RichTextElementType `json:"type"`
		}
		if err := json.Unmarshal(r, &s); err != nil {
			return err
		}
		var elem RichTextElement
		switch s.Type {
		case RTESection:
			elem = &RichTextSection{}
		case RTEList:
			elem = &RichTextList{}
		case RTEQuote:
			elem = &RichTextQuote{}
		case RTEPreformatted:
			elem = &RichTextPreformatted{}
		default:
			elems = append(elems, &RichTextUnknown{
				Type: s.Type,
				Raw:  string(r),
			})
			continue
		}
		if err := json.Unmarshal(r, &elem); err != nil {
			return err
		}
		elems = append(elems, elem)
	}
	*e = RichTextBlock{
		Type:     raw.Type,
		BlockID:  raw.BlockID,
		Elements: elems,
	}
	return nil
}

// NewRichTextBlock returns a new instance of RichText Block.
func NewRichTextBlock(blockID string, elements ...RichTextElement) *RichTextBlock {
	return &RichTextBlock{
		Type:     MBTRichText,
		BlockID:  blockID,
		Elements: elements,
	}
}

type RichTextElementType string

type RichTextElement interface {
	RichTextElementType() RichTextElementType
}

const (
	RTEList         RichTextElementType = "rich_text_list"
	RTEPreformatted RichTextElementType = "rich_text_preformatted"
	RTEQuote        RichTextElementType = "rich_text_quote"
	RTESection      RichTextElementType = "rich_text_section"
	RTEUnknown      RichTextElementType = "rich_text_unknown"
)

type RichTextUnknown struct {
	Type RichTextElementType
	Raw  string
}

func (u RichTextUnknown) RichTextElementType() RichTextElementType {
	return u.Type
}

type RichTextListElementType string

const (
	RTEListOrdered RichTextListElementType = "ordered"
	RTEListBullet  RichTextListElementType = "bullet"
)

type RichTextList struct {
	Type     RichTextElementType     `json:"type"`
	Elements []RichTextElement       `json:"elements"`
	Style    RichTextListElementType `json:"style"`
	Indent   int                     `json:"indent"`
	Border   int                     `json:"border"`
	Offset   int                     `json:"offset"`
}

// NewRichTextList returns a new rich text list element.
func NewRichTextList(style RichTextListElementType, indent int, elements ...RichTextElement) *RichTextList {
	return &RichTextList{
		Type:     RTEList,
		Elements: elements,
		Style:    style,
		Indent:   indent,
	}
}

// ElementType returns the type of the Element
func (s RichTextList) RichTextElementType() RichTextElementType {
	return s.Type
}

func (e *RichTextList) UnmarshalJSON(b []byte) error {
	var raw struct {
		RawElements []json.RawMessage       `json:"elements"`
		Style       RichTextListElementType `json:"style"`
		Indent      int                     `json:"indent"`
		Border      int                     `json:"border"`
		Offset      int                     `json:"offset"`
	}
	if string(b) == "{}" {
		return nil
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	elems := make([]RichTextElement, 0, len(raw.RawElements))
	for _, r := range raw.RawElements {
		var s struct {
			Type RichTextElementType `json:"type"`
		}
		if err := json.Unmarshal(r, &s); err != nil {
			return err
		}
		var elem RichTextElement
		switch s.Type {
		case RTESection:
			elem = &RichTextSection{}
		case RTEList:
			elem = &RichTextList{}
		case RTEQuote:
			elem = &RichTextQuote{}
		case RTEPreformatted:
			elem = &RichTextPreformatted{}
		default:
			elems = append(elems, &RichTextUnknown{
				Type: s.Type,
				Raw:  string(r),
			})
			continue
		}
		if err := json.Unmarshal(r, elem); err != nil {
			return err
		}
		elems = append(elems, elem)
	}
	*e = RichTextList{
		Type:     RTEList,
		Elements: elems,
		Style:    raw.Style,
		Indent:   raw.Indent,
		Border:   raw.Border,
		Offset:   raw.Offset,
	}
	return nil
}

type RichTextSection struct {
	Type     RichTextElementType      `json:"type"`
	Elements []RichTextSectionElement `json:"elements"`
}

// RichTextElementType returns the type of the Element
func (s RichTextSection) RichTextElementType() RichTextElementType {
	return s.Type
}

func (e *RichTextSection) UnmarshalJSON(b []byte) error {
	var raw struct {
		RawElements []json.RawMessage   `json:"elements"`
		Type        RichTextElementType `json:"type"`
	}
	if string(b) == "{}" {
		return nil
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	elems := make([]RichTextSectionElement, 0, len(raw.RawElements))
	for _, r := range raw.RawElements {
		var s struct {
			Type RichTextSectionElementType `json:"type"`
		}
		if err := json.Unmarshal(r, &s); err != nil {
			return err
		}
		var elem RichTextSectionElement
		switch s.Type {
		case RTSEText:
			elem = &RichTextSectionTextElement{}
		case RTSEChannel:
			elem = &RichTextSectionChannelElement{}
		case RTSEUser:
			elem = &RichTextSectionUserElement{}
		case RTSEEmoji:
			elem = &RichTextSectionEmojiElement{}
		case RTSELink:
			elem = &RichTextSectionLinkElement{}
		case RTSETeam:
			elem = &RichTextSectionTeamElement{}
		case RTSEUserGroup:
			elem = &RichTextSectionUserGroupElement{}
		case RTSEDate:
			elem = &RichTextSectionDateElement{}
		case RTSEBroadcast:
			elem = &RichTextSectionBroadcastElement{}
		case RTSEColor:
			elem = &RichTextSectionColorElement{}
		default:
			elems = append(elems, &RichTextSectionUnknownElement{
				Type: s.Type,
				Raw:  string(r),
			})
			continue
		}
		if err := json.Unmarshal(r, elem); err != nil {
			return err
		}
		elems = append(elems, elem)
	}
	if raw.Type == "" {
		raw.Type = RTESection
	}
	*e = RichTextSection{
		Type:     raw.Type,
		Elements: elems,
	}
	return nil
}

// NewRichTextSectionBlockElement .
func NewRichTextSection(elements ...RichTextSectionElement) *RichTextSection {
	return &RichTextSection{
		Type:     RTESection,
		Elements: elements,
	}
}

type RichTextSectionElementType string

const (
	RTSEBroadcast RichTextSectionElementType = "broadcast"
	RTSEChannel   RichTextSectionElementType = "channel"
	RTSEColor     RichTextSectionElementType = "color"
	RTSEDate      RichTextSectionElementType = "date"
	RTSEEmoji     RichTextSectionElementType = "emoji"
	RTSELink      RichTextSectionElementType = "link"
	RTSETeam      RichTextSectionElementType = "team"
	RTSEText      RichTextSectionElementType = "text"
	RTSEUser      RichTextSectionElementType = "user"
	RTSEUserGroup RichTextSectionElementType = "usergroup"

	RTSEUnknown RichTextSectionElementType = "unknown"
)

type RichTextSectionElement interface {
	RichTextSectionElementType() RichTextSectionElementType
}

type RichTextSectionTextStyle struct {
	Bold   bool `json:"bold,omitempty"`
	Italic bool `json:"italic,omitempty"`
	Strike bool `json:"strike,omitempty"`
	Code   bool `json:"code,omitempty"`
}

type RichTextSectionTextElement struct {
	Type  RichTextSectionElementType `json:"type"`
	Text  string                     `json:"text"`
	Style *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionTextElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionTextElement(text string, style *RichTextSectionTextStyle) *RichTextSectionTextElement {
	return &RichTextSectionTextElement{
		Type:  RTSEText,
		Text:  text,
		Style: style,
	}
}

type RichTextSectionChannelElement struct {
	Type      RichTextSectionElementType `json:"type"`
	ChannelID string                     `json:"channel_id"`
	Style     *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionChannelElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionChannelElement(channelID string, style *RichTextSectionTextStyle) *RichTextSectionChannelElement {
	return &RichTextSectionChannelElement{
		Type:      RTSEText,
		ChannelID: channelID,
		Style:     style,
	}
}

type RichTextSectionUserElement struct {
	Type   RichTextSectionElementType `json:"type"`
	UserID string                     `json:"user_id"`
	Style  *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionUserElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionUserElement(userID string, style *RichTextSectionTextStyle) *RichTextSectionUserElement {
	return &RichTextSectionUserElement{
		Type:   RTSEUser,
		UserID: userID,
		Style:  style,
	}
}

type RichTextSectionEmojiElement struct {
	Type     RichTextSectionElementType `json:"type"`
	Name     string                     `json:"name"`
	SkinTone int                        `json:"skin_tone,omitempty"`
	Unicode  string                     `json:"unicode,omitempty"`
	Style    *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionEmojiElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionEmojiElement(name string, skinTone int, style *RichTextSectionTextStyle) *RichTextSectionEmojiElement {
	return &RichTextSectionEmojiElement{
		Type:     RTSEEmoji,
		Name:     name,
		SkinTone: skinTone,
		Style:    style,
	}
}

type RichTextSectionLinkElement struct {
	Type  RichTextSectionElementType `json:"type"`
	URL   string                     `json:"url"`
	Text  string                     `json:"text,omitempty"`
	Style *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionLinkElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionLinkElement(url, text string, style *RichTextSectionTextStyle) *RichTextSectionLinkElement {
	return &RichTextSectionLinkElement{
		Type:  RTSELink,
		URL:   url,
		Text:  text,
		Style: style,
	}
}

type RichTextSectionTeamElement struct {
	Type   RichTextSectionElementType `json:"type"`
	TeamID string                     `json:"team_id"`
	Style  *RichTextSectionTextStyle  `json:"style,omitempty"`
}

func (r RichTextSectionTeamElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionTeamElement(teamID string, style *RichTextSectionTextStyle) *RichTextSectionTeamElement {
	return &RichTextSectionTeamElement{
		Type:   RTSETeam,
		TeamID: teamID,
		Style:  style,
	}
}

type RichTextSectionUserGroupElement struct {
	Type        RichTextSectionElementType `json:"type"`
	UsergroupID string                     `json:"usergroup_id"`
}

func (r RichTextSectionUserGroupElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionUserGroupElement(usergroupID string) *RichTextSectionUserGroupElement {
	return &RichTextSectionUserGroupElement{
		Type:        RTSEUserGroup,
		UsergroupID: usergroupID,
	}
}

type RichTextSectionDateElement struct {
	Type      RichTextSectionElementType `json:"type"`
	Timestamp JSONTime                   `json:"timestamp"`
	Format    string                     `json:"format"`
	URL       *string                    `json:"url,omitempty"`
	Fallback  *string                    `json:"fallback,omitempty"`
}

func (r RichTextSectionDateElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionDateElement(timestamp int64, format string, url *string, fallback *string) *RichTextSectionDateElement {
	return &RichTextSectionDateElement{
		Type:      RTSEDate,
		Timestamp: JSONTime(timestamp),
		Format:    format,
		URL:       url,
		Fallback:  fallback,
	}
}

type RichTextSectionBroadcastElement struct {
	Type  RichTextSectionElementType `json:"type"`
	Range string                     `json:"range"`
}

func (r RichTextSectionBroadcastElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionBroadcastElement(rangeStr string) *RichTextSectionBroadcastElement {
	return &RichTextSectionBroadcastElement{
		Type:  RTSEBroadcast,
		Range: rangeStr,
	}
}

type RichTextSectionColorElement struct {
	Type  RichTextSectionElementType `json:"type"`
	Value string                     `json:"value"`
}

func (r RichTextSectionColorElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

func NewRichTextSectionColorElement(value string) *RichTextSectionColorElement {
	return &RichTextSectionColorElement{
		Type:  RTSEColor,
		Value: value,
	}
}

type RichTextSectionUnknownElement struct {
	Type RichTextSectionElementType `json:"type"`
	Raw  string
}

func (r RichTextSectionUnknownElement) RichTextSectionElementType() RichTextSectionElementType {
	return r.Type
}

// RichTextQuote represents rich_text_quote element type.
type RichTextQuote RichTextSection

// RichTextElementType returns the type of the Element
func (s *RichTextQuote) RichTextElementType() RichTextElementType {
	return s.Type
}

func (s *RichTextQuote) UnmarshalJSON(b []byte) error {
	// reusing the RichTextSection struct, as it's the same as RichTextQuote.
	var rts RichTextSection
	if err := json.Unmarshal(b, &rts); err != nil {
		return err
	}
	*s = RichTextQuote(rts)
	s.Type = RTEQuote
	return nil
}

// RichTextPreformatted represents rich_text_quote element type.
type RichTextPreformatted struct {
	RichTextSection
	Border int `json:"border"`
}

// RichTextElementType returns the type of the Element
func (s *RichTextPreformatted) RichTextElementType() RichTextElementType {
	return s.Type
}

func (s *RichTextPreformatted) UnmarshalJSON(b []byte) error {
	var rts RichTextSection
	if err := json.Unmarshal(b, &rts); err != nil {
		return err
	}
	// we define standalone fields because we need to unmarshal the border
	// field.  We can not directly unmarshal the data into
	// RichTextPreformatted because it will cause an infinite loop.  We also
	// can not define a struct with embedded RichTextSection and Border fields
	// because the json package will not unmarshal the data into the
	// standalone fields, once it sees UnmarshalJSON method on the embedded
	// struct.  The drawback is that we have to process the data twice, and
	// have to define a standalone struct with the same set of fields as the
	// original struct, which may become a maintenance burden (i.e. update the
	// fields in two places, should it ever change).
	var standalone struct {
		Border int `json:"border"`
	}
	if err := json.Unmarshal(b, &standalone); err != nil {
		return err
	}
	*s = RichTextPreformatted{
		RichTextSection: rts,
		Border:          standalone.Border,
	}
	s.Type = RTEPreformatted
	return nil
}
