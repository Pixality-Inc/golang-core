package pusher

import (
	"maps"

	"github.com/pixality-inc/golang-core/util"
)

//nolint:modernize
type Message interface {
	Title() string
	Subtitle() string
	Body() string
	Badges() int
	Silent() bool
	CustomData() map[string]any
}

type MessageImpl struct {
	TitleValue      *string
	SubtitleValue   *string
	BodyValue       *string
	BadgesValue     *int
	SilentValue     *bool
	CustomDataValue map[string]any
}

func NewMessage() *MessageImpl {
	return &MessageImpl{
		CustomDataValue: make(map[string]any),
	}
}

func (m *MessageImpl) SetTitle(title string) *MessageImpl {
	m.TitleValue = &title

	return m
}

func (m *MessageImpl) SetSubtitle(subtitle string) *MessageImpl {
	m.SubtitleValue = &subtitle

	return m
}

func (m *MessageImpl) SetBody(body string) *MessageImpl {
	m.BodyValue = &body

	return m
}

func (m *MessageImpl) SetBadges(badges int) *MessageImpl {
	m.BadgesValue = &badges

	return m
}

func (m *MessageImpl) SetSilent(silent bool) *MessageImpl {
	m.SilentValue = &silent

	return m
}

func (m *MessageImpl) AddCustomDataMap(values map[string]any) *MessageImpl {
	if m.CustomDataValue == nil {
		m.CustomDataValue = make(map[string]any)
	}

	maps.Copy(m.CustomDataValue, values)

	return m
}

func (m *MessageImpl) AddCustomData(key string, value any) *MessageImpl {
	if m.CustomDataValue == nil {
		m.CustomDataValue = make(map[string]any)
	}

	m.CustomDataValue[key] = value

	return m
}

func (m *MessageImpl) Title() string {
	return util.OrDefault(m.TitleValue, "")
}

func (m *MessageImpl) Subtitle() string {
	return util.OrDefault(m.SubtitleValue, "")
}

func (m *MessageImpl) Body() string {
	return util.OrDefault(m.BodyValue, "")
}

func (m *MessageImpl) Badges() int {
	return util.OrDefault(m.BadgesValue, 0)
}

func (m *MessageImpl) Silent() bool {
	return util.OrDefault(m.SilentValue, false)
}

func (m *MessageImpl) CustomData() map[string]any {
	return m.CustomDataValue
}
