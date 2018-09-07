package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"

	"github.com/go-messaging-service/goms-server/src/config"
	"github.com/go-messaging-service/goms-server/src/msg"
	"github.com/go-messaging-service/goms-server/src/util"
	"github.com/hauke96/sigolo"
)

type Handler struct {
	connection       *net.Conn
	connectionClosed bool
	config           *config.Config
	registeredTopics []string
	SendEvent        []func(Handler, *msg.Message)
	ErrorEvent       []func(*Handler, string, string)
}

const MAX_PRINTING_LENGTH int = 80

// Init initializes the handler with the given connection.
func (ch *Handler) Init(connection *net.Conn, config *config.Config) {
	ch.connection = connection
	ch.config = config
}

// HandleConnection starts a routine to handle registration and sending messages.
// This will run until the client logs out, so run this in a goroutine.
func (ch *Handler) HandleConnection() {
	// Not initialized
	if ch.connection == nil {
		sigolo.Fatal("Connection not set!")
	}

	messageTypes := []string{msg.MT_REGISTER,
		msg.MT_LOGOUT,
		msg.MT_CLOSE,
		msg.MT_SEND}

	handler := []func(msg.Message){ch.handleRegistration,
		ch.handleLogout,
		ch.handleClose,
		ch.handleSending}

	reader := bufio.NewReader(*ch.connection)

	// Now a arbitrary amount of registration, logout, close and send messages is allowed
	for !ch.connectionClosed {
		ch.waitFor(
			messageTypes,
			handler,
			reader)
	}

	// TODO fire closed-event so that the distributor can react on that
}

// waitFor wats until on of the given message types arrived.
// The i-th argument in the messageTypes array must match to the i-th argument in the handler array.
func (ch *Handler) waitFor(messageTypes []string, handler []func(message msg.Message), reader *bufio.Reader) {

	// Check if the arrays match and error/fatal here
	if len(messageTypes) != len(handler) {
		if len(messageTypes) > len(handler) {
			// Fatal here to prevent a "slice bounds out of range" error during runtime
			sigolo.Fatal("There're more defined message types then functions mapped to them.")
		} else {
			sigolo.Error("There're more defined functions then message types here. Some message types might not be covered. Fix that!")
		}
	}

	rawMessage, err := reader.ReadString('\n')

	if err == nil {
		// the length of the message that should be printed
		maxOutputLength := int(math.Min(float64(len(rawMessage))-1, float64(MAX_PRINTING_LENGTH)))
		output := rawMessage[:maxOutputLength]
		if MAX_PRINTING_LENGTH < len(rawMessage)-1 {
			output += " [...]"
		}
		sigolo.Info(output)

		// JSON to Message-struct
		message := getMessageFromJSON(rawMessage)

		// check type
		for i := 0; i < len(messageTypes); i++ {
			messageType := messageTypes[i]
			sigolo.Debug("Check if received type '" + message.Messagetype + "' is type '" + messageType + "'")

			if message.Messagetype == messageType {
				sigolo.Debug("Handle " + messageType + " type")
				handler[i](message)
				break
			}
		}
	} else {
		sigolo.Info("The connection will be closed. Reason: " + err.Error())
		ch.exit()
		ch.connectionClosed = true
	}
}

// getMessageFromJSON converts the given json-data into a message object.
func getMessageFromJSON(jsonData string) msg.Message {
	message := msg.Message{}
	json.Unmarshal([]byte(jsonData), &message)
	return message
}

// handleRegistration registeres this connection to the topics specified in the message.
func (ch *Handler) handleRegistration(message msg.Message) {
	sigolo.Debug("Try to register to topics " + fmt.Sprintf("%#v", message.Topics))

	// A comma separated list of all topics, the client is not allowed to register to
	forbiddenTopics := ""
	alreadyRegisteredTopics := ""

	for _, topic := range message.Topics {
		//TODO create a service for this. This should later take care of different user rights
		if !util.ContainsString(ch.config.TopicConfig.Topics, topic) {
			forbiddenTopics += topic + ","
			sigolo.Info("Clients wants to register on invalid topic (" + topic + ").")

		} else if util.ContainsString(ch.registeredTopics, topic) {
			alreadyRegisteredTopics += topic + ","
			sigolo.Debug("Client already registered on " + topic)

		} else {
			ch.registeredTopics = append(ch.registeredTopics, topic)
			sigolo.Debug("Register " + topic)
		}
	}

	// Send error message for forbidden topics and cut trailing comma
	if len(forbiddenTopics) != 0 {
		forbiddenTopics = strings.TrimSuffix(forbiddenTopics, ",")

		for _, event := range ch.ErrorEvent {
			event(ch, msg.ERR_REG_INVALID_TOPIC, forbiddenTopics)
		}
	}

	// Send error message for already registered topics and cut trailing comma
	if len(alreadyRegisteredTopics) != 0 {
		alreadyRegisteredTopics = strings.TrimSuffix(alreadyRegisteredTopics, ",")

		for _, event := range ch.ErrorEvent {
			event(ch, msg.ERR_REG_ALREADY_REGISTERED, alreadyRegisteredTopics)
		}
	}
}

// handleSending send the given message to all clients interested in the topics specified in the message.
func (ch *Handler) handleSending(message msg.Message) {
	for _, event := range ch.SendEvent {
		event(*ch, &message)
	}
}

// handleLogout logs the client out.
func (ch *Handler) handleLogout(message msg.Message) {
	sigolo.Debug(fmt.Sprintf("Unsubscribe from topics %#v", message.Topics))
	ch.logout(message.Topics)
}

// handleClose logs the client out from all topics and closes the connection.
func (ch *Handler) handleClose(message msg.Message) {
	ch.exit()
}

// exit logs the client out from all topics and closes the connection.
func (ch *Handler) exit() {
	sigolo.Debug("Unsubscribe from all topics")
	ch.logout(ch.registeredTopics)

	sigolo.Debug("Close connection")
	(*ch.connection).Close()
	ch.connectionClosed = true
}

// logout will logs the client out from the given topics.
func (ch *Handler) logout(topics []string) {
	for _, topic := range topics {
		ch.registeredTopics = util.RemoveString(ch.registeredTopics, topic)
	}

	ch.registeredTopics = util.RemoveStrings(ch.registeredTopics, topics)
}

func (ch *Handler) IsRegisteredTo(topic string) bool {
	return util.ContainsString(ch.registeredTopics, topic)
}
