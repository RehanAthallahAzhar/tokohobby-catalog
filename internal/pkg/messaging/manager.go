package messaging

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type Manager struct {
	channel      *amqp.Channel
	inputChan    chan NotificationPayload
	exchangeName string
}

func NewManager(amqpConn *amqp.Connection, bufferSize int) (*Manager, error) {
	ch, err := amqpConn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare Exchange agar aman (idempotent)
	// Gunakan nama exchange unik global, misal "tokohobby.events"
	err = ch.ExchangeDeclare("tokohobby.events", "topic", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	mgr := &Manager{
		channel:      ch,
		inputChan:    make(chan NotificationPayload, bufferSize),
		exchangeName: "tokohobby.events",
	}

	// Jalankan worker background
	go mgr.worker()

	return mgr, nil
}

// worker memindahkan data dari memory (Channel) ke RabbitMQ
func (m *Manager) worker() {
	for payload := range m.inputChan {
		body, err := json.Marshal(payload)
		if err != nil {
			log.Printf("Messaging Marshal Error: %v", err)
			continue
		}

		err = m.channel.Publish(
			m.exchangeName,
			string(payload.Type), // Routing Key
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        body,
			},
		)

		if err != nil {
			log.Printf("RabbitMQ Publish Error: %v", err)
		}
	}
}

// Send (Public Method) - Non blocking!
func (m *Manager) Send(payload NotificationPayload) {
	select {
	case m.inputChan <- payload:
		// Masuk antrian memory
	default:
		// Antrian penuh, drop biar ga bikin lemot user request
		log.Println("Messaging Buffer Full! Dropped event:", payload.Type)
	}
}

func (m *Manager) Close() {
	close(m.inputChan)
	m.channel.Close()
}
