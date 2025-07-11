// common/status_messages.go

package common

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"sync"

	"MetaRekordFixer/theme"
)

// MessageType defines the type of status message
type MessageType string

// Deprecated: Will be removed after new error handling implementation is complete.
// Use SeverityXXX constants from severity.go instead.
const (
	MessageInfo     MessageType = MessageType(SeverityInfo)     // Use SeverityInfo instead
	MessageWarning  MessageType = MessageType(SeverityWarning)  // Use SeverityWarning instead
	MessageError    MessageType = MessageType(SeverityError)    // Use SeverityError instead
	MessageCritical MessageType = MessageType(SeverityCritical) // Use SeverityCritical instead
)

// StatusMessage represents a single status message with its type and content
type StatusMessage struct {
	Type    MessageType
	Content string
}

// StatusMessagesContainer is a widget that displays status messages with icons
type StatusMessagesContainer struct {
	widget.BaseWidget
	messages  []StatusMessage
	container *fyne.Container
	scroll    *container.Scroll
	mutex     sync.Mutex
}

// NewStatusMessagesContainer creates a new status messages container
func NewStatusMessagesContainer() *StatusMessagesContainer {
	smc := &StatusMessagesContainer{
		messages: []StatusMessage{},
	}
	smc.ExtendBaseWidget(smc)
	smc.container = container.NewVBox()
	smc.scroll = container.NewScroll(smc.container)

	// Set minimum size for the scroll container in case of 700px height of main window
	smc.scroll.SetMinSize(fyne.NewSize(0, 400))
	return smc
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer
func (smc *StatusMessagesContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(smc.scroll)
}

// AddMessage adds a new message to the container
func (smc *StatusMessagesContainer) AddMessage(messageType MessageType, content string) {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	// Add message to the internal slice
	smc.messages = append(smc.messages, StatusMessage{Type: messageType, Content: content})

	// Create message row with icon
	var icon fyne.Resource

	switch messageType {
	case MessageInfo:
		icon = theme.InfoIcon()
	case MessageWarning:
		icon = theme.WarningIcon()
	case MessageError, MessageCritical:
		icon = theme.ErrorIcon()
	}

	// Create label with the message content
	messageLabel := widget.NewLabel(content)
	messageLabel.Alignment = fyne.TextAlignLeading
	messageLabel.TextStyle.Bold = messageType != MessageInfo // Bold for warnings, errors and critical errors

	// Create a smaller icon with fixed size
	iconWidget := widget.NewIcon(icon)
	// Create a container with fixed size for the icon
	iconContainer := container.New(
		layout.NewCenterLayout(), // Center the icon in the container
		iconWidget)
	// Use fixed size for more compact display
	iconContainer.Resize(fyne.NewSize(16, 16))

	// Create row with icon and message
	row := container.NewHBox(
		iconContainer,
		messageLabel,
	)

	// Add to the container
	smc.container.Add(row)

	// Refresh the widget
	smc.Refresh()
}

// AddInfoMessage adds an information message
func (smc *StatusMessagesContainer) AddInfoMessage(content string) {
	smc.AddMessage(MessageInfo, content)
}

// AddWarningMessage adds a warning message
func (smc *StatusMessagesContainer) AddWarningMessage(content string) {
	smc.AddMessage(MessageWarning, content)
}

// AddErrorMessage adds an error message
func (smc *StatusMessagesContainer) AddErrorMessage(content string) {
	smc.AddMessage(MessageError, content)
}

// AddCriticalMessage adds a critical error message
func (smc *StatusMessagesContainer) AddCriticalMessage(content string) {
	smc.AddMessage(MessageCritical, content)
}

// ClearMessages removes all messages from the container
func (smc *StatusMessagesContainer) ClearMessages() {
	smc.mutex.Lock()
	defer smc.mutex.Unlock()
	smc.messages = []StatusMessage{}
	smc.container.RemoveAll()
	smc.Refresh()
}


