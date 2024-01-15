package contextdb // import github.com/chrlic/otelcol-cust/collector/receiver/ciscoaci/db

import (
	"fmt"
	"sync"
	"time"
)

type ContextProviderExtension interface {
	SubscribeToContext(subscriberId string, topic string) (*ContextSubscriber, error)
	UnsubscribeContext(subscriberId string, topic string)
}

type ContextBus struct {
	channelBufferSize int
	topics            map[string][]*ContextSubscriber

	mutex sync.Mutex
}

type ContextSubscriber struct {
	Id string
	ch chan ContextData
}

func ContextBusFactory(channelBufferSize int) *ContextBus {
	c := ContextBus{
		channelBufferSize: channelBufferSize,
		topics:            map[string][]*ContextSubscriber{},
		mutex:             sync.Mutex{},
	}

	return &c
}

func (c *ContextBus) HasTopic(topic string) bool {
	_, ok := c.topics[topic]
	return ok
}

func (c *ContextBus) CreateTopic(topic string) {
	if !c.HasTopic(topic) {
		c.topics[topic] = []*ContextSubscriber{}
	}
}

func (c *ContextBus) DeleteTopic(topic string) {
	if c.HasTopic(topic) {
		delete(c.topics, topic)
	}
}

func (c *ContextBus) IsSubscribedToTopic(subscriberId string, topic string) bool {

	if !c.HasTopic(topic) {
		return false
	}

	subs := c.topics[topic]
	for _, sub := range subs {
		if sub.Id == subscriberId {
			return true
		}
	}

	return false
}

func (c *ContextBus) SubscribeToTopic(subscriberId string, topic string) (*ContextSubscriber, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.HasTopic(topic) {
		return nil, fmt.Errorf("Topic %s does not exist", topic)
	}

	subs := c.topics[topic]
	for _, sub := range subs {
		if sub.Id == subscriberId {
			return sub, nil
		}
	}

	subscriber := ContextSubscriber{
		Id: subscriberId,
		ch: make(chan ContextData, c.channelBufferSize),
	}
	subs = append(subs, &subscriber)
	c.topics[topic] = subs

	return &subscriber, nil
}

func (c *ContextBus) UnsubscribeFromTopic(subscriberId string, topic string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.HasTopic(topic) {
		return fmt.Errorf("Topic %s does not exist", topic)
	}

	subs := c.topics[topic]
	tempSubs := subs[:0]
	for _, sub := range subs {
		if sub.Id != subscriberId {
			tempSubs = append(tempSubs, sub)
		} else {
			close(sub.ch)
		}
	}
	c.topics[topic] = tempSubs

	return nil
}

func (c *ContextBus) PublishOnTopic(topic string, data ContextData) {
	if !c.HasTopic(topic) {
		return
	}

	subs := c.topics[topic]
	for _, sub := range subs {
		// non-blocking write to channel
		// !!!! if subscriber does not pick up, data is lost !!!!
		select {
		case sub.ch <- data:
		default:
		}
	}
}

func (s *ContextSubscriber) ReadData(timeout int) (*ContextData, bool, bool) { // false if timeout or error

	if timeout == 0 { // non-blocking read
		select {
		case data, closed := <-s.ch:
			return &data, true, closed
		default:
			return nil, false, false
		}
	}

	if timeout < 0 { // block until data received
		data, closed := <-s.ch
		return &data, true, closed
	} else { // block until timeout
		select {
		case data, closed := <-s.ch:
			return &data, true, closed
		case <-time.After(time.Duration(timeout) * time.Millisecond):
			return nil, false, false
		}
	}
}

func (s *ContextSubscriber) HandleMessages(handler func(*ContextData)) {
	go func() {
		for {
			data, _, closed := s.ReadData(-1)
			if closed {
				break
			}

			handler(data)
		}
	}()
}

func (s *ContextSubscriber) AttachContextDb(db ContextDb, table string) {
	s.HandleMessages(func(data *ContextData) {
		record := ContextRecord{
			Data:              *data,
			LastUpdatedMillis: time.Now().UnixMilli(),
		}
		db.InsertOrUpdateRecord(table, &record)
	})
}
