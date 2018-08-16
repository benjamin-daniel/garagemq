package exchange

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/valinurovam/garagemq/amqp"
	"github.com/valinurovam/garagemq/binding"
)

// available exchange types
const (
	ExTypeDirect    = iota + 1
	ExTypeFanout
	ExTypeTopic
	ExTypeHeaders
)

var exchangeTypeIDAliasMap = map[byte]string{
	ExTypeDirect:  "direct",
	ExTypeFanout:  "fanout",
	ExTypeTopic:   "topic",
	ExTypeHeaders: "headers",
}

var exchangeTypeAliasIDMap = map[string]byte{
	"direct":  ExTypeDirect,
	"fanout":  ExTypeFanout,
	"topic":   ExTypeTopic,
	"headers": ExTypeHeaders,
}

// Exchange implements AMQP-exchange
type Exchange struct {
	Name       string
	exType     byte
	durable    bool
	autoDelete bool
	internal   bool
	system     bool
	arguments  *amqp.Table
	bindLock   sync.Mutex
	bindings   []*binding.Binding
}

// GetExchangeTypeAlias returns exchange type alias by id
func GetExchangeTypeAlias(id byte) (alias string, err error) {
	if alias, ok := exchangeTypeIDAliasMap[id]; ok {
		return alias, nil
	}
	return "", fmt.Errorf("undefined exchange type '%d'", id)
}

// GetExchangeTypeID returns exchange type id by alias
func GetExchangeTypeID(alias string) (id byte, err error) {
	if id, ok := exchangeTypeAliasIDMap[alias]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("undefined exchange alias '%s'", alias)
}

// NewExchange returns new instance of Exchange
func NewExchange(name string, exType byte, durable bool, autoDelete bool, internal bool, system bool) *Exchange {
	return &Exchange{
		Name:       name,
		exType:     exType,
		durable:    durable,
		autoDelete: autoDelete,
		internal:   internal,
		system:     system,
	}
}

// AppendBinding check and append binding
// method check if binding already exists and ignore it
func (ex *Exchange) AppendBinding(newBind *binding.Binding) {
	ex.bindLock.Lock()
	defer ex.bindLock.Unlock()
	for _, bind := range ex.bindings {
		if bind.Equal(newBind) {
			return
		}
	}
	ex.bindings = append(ex.bindings, newBind)
}

// RemoveBinding remove binding
func (ex *Exchange) RemoveBinding(rmBind *binding.Binding) {
	ex.bindLock.Lock()
	defer ex.bindLock.Unlock()
	for i, bind := range ex.bindings {
		if bind.Equal(rmBind) {
			ex.bindings = append(ex.bindings[:i], ex.bindings[i+1:]...)
			return
		}
	}
}

// RemoveQueueBindings remove bindings for queue and return removed bindings
func (ex *Exchange) RemoveQueueBindings(queueName string) []*binding.Binding {
	var newBindings []*binding.Binding
	var removedBindings []*binding.Binding
	ex.bindLock.Lock()
	defer ex.bindLock.Unlock()
	for _, bind := range ex.bindings {
		if bind.GetQueue() != queueName {
			newBindings = append(newBindings, bind)
		} else {
			removedBindings = append(removedBindings, bind)
		}
	}

	ex.bindings = newBindings
	return removedBindings
}

// GetMatchedQueues returns queues matched for message routing key
func (ex *Exchange) GetMatchedQueues(message *amqp.Message) (matchedQueues map[string]bool) {
	matchedQueues = make(map[string]bool)
	switch ex.exType {
	case ExTypeDirect:
		for _, bind := range ex.bindings {
			if bind.MatchDirect(message.Exchange, message.RoutingKey) {
				matchedQueues[bind.GetQueue()] = true
				return
			}
		}
	case ExTypeFanout:
		for _, bind := range ex.bindings {
			if bind.MatchFanout(message.Exchange) {
				matchedQueues[bind.GetQueue()] = true
			}
		}
	case ExTypeTopic:
		for _, bind := range ex.bindings {
			if bind.MatchTopic(message.Exchange, message.RoutingKey) {
				matchedQueues[bind.GetQueue()] = true
			}
		}
	}
	return
}

// EqualWithErr returns is given exchange equal to current
func (ex *Exchange) EqualWithErr(exB *Exchange) error {
	errTemplate := "inequivalent arg '%s' for exchange '%s': received '%s' but current is '%s'"
	if ex.exType != exB.ExType() {
		aliasA, _ := GetExchangeTypeAlias(ex.exType)
		aliasB, _ := GetExchangeTypeAlias(exB.ExType())
		return fmt.Errorf(
			errTemplate,
			"type",
			ex.Name,
			aliasB,
			aliasA,
		)
	}
	if ex.durable != exB.IsDurable() {
		return fmt.Errorf(errTemplate, "durable", ex.Name, exB.IsDurable(), ex.durable)
	}
	if ex.autoDelete != exB.IsAutoDelete() {
		return fmt.Errorf(errTemplate, "autoDelete", ex.Name, exB.IsAutoDelete(), ex.autoDelete)
	}
	if ex.internal != exB.IsInternal() {
		return fmt.Errorf(errTemplate, "internal", ex.Name, exB.IsInternal(), ex.internal)
	}
	return nil
}

// GetBindings returns exchange's bindings
func (ex *Exchange) GetBindings() []*binding.Binding {
	ex.bindLock.Lock()
	defer ex.bindLock.Unlock()
	return ex.bindings
}

// IsDurable returns is exchange durable
func (ex *Exchange) IsDurable() bool {
	return ex.durable
}

// IsSystem returns is exchange system
func (ex *Exchange) IsSystem() bool {
	return ex.system
}

// IsAutoDelete returns should be exchange deleted when all queues have finished using it
func (ex *Exchange) IsAutoDelete() bool {
	return ex.autoDelete
}

// IsInternal returns that the exchange may not be used directly by publishers,
// but only when bound to other exchanges
func (ex *Exchange) IsInternal() bool {
	return ex.internal
}

// Marshal returns raw representation of exchange to store into storage
func (ex *Exchange) Marshal(protoVersion string) []byte {
	buf := bytes.NewBuffer(make([]byte, 0))
	amqp.WriteShortstr(buf, ex.Name)
	amqp.WriteOctet(buf, ex.exType)
	return buf.Bytes()
}

// Unmarshal returns exchange from storage raw bytes data
func (ex *Exchange) Unmarshal(data []byte) {
	buf := bytes.NewReader(data)
	ex.Name, _ = amqp.ReadShortstr(buf)
	ex.exType, _ = amqp.ReadOctet(buf)
	ex.durable = true
}

// GetName returns exchange name
func (ex *Exchange) GetName() string {
	return ex.Name
}

// ExType returns exchange type
func (ex *Exchange) ExType() byte {
	return ex.exType
}
