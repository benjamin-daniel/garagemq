package server

import (
	"github.com/valinurovam/garagemq/amqp"
	"github.com/valinurovam/garagemq/exchange"
	"fmt"
	"strings"
)

func (channel *Channel) exchangeRoute(method amqp.Method) *amqp.Error {
	switch method := method.(type) {
	case *amqp.ExchangeDeclare:
		return channel.exchangeDeclare(method)
	}

	return amqp.NewConnectionError(amqp.NotImplemented, "unable to route queue method "+method.Name(), method.ClassIdentifier(), method.MethodIdentifier())
}

func (channel *Channel) exchangeDeclare(method *amqp.ExchangeDeclare) *amqp.Error {
	exTypeId, err := exchange.GetExchangeTypeId(method.Type)
	if err != nil {
		return amqp.NewChannelError(amqp.NotImplemented, err.Error(), method.ClassIdentifier(), method.MethodIdentifier())
	}

	if method.Exchange == "" {
		return amqp.NewChannelError(
			amqp.CommandInvalid,
			"exchange name is requred",
			method.ClassIdentifier(),
			method.MethodIdentifier(),
		)
	}

	existingExchange := channel.conn.getVirtualHost().GetExchange(method.Exchange)
	if method.Passive {
		if method.NoWait {
			return nil
		}

		if existingExchange == nil {
			return amqp.NewChannelError(
				amqp.NotFound,
				fmt.Sprintf("exchange '%s' not found", method.Exchange),
				method.ClassIdentifier(),
				method.MethodIdentifier(),
			)
		} else {
			channel.SendMethod(&amqp.ExchangeDeclareOk{})
		}

		return nil
	}

	if strings.HasPrefix(method.Exchange, "amq.") {
		return amqp.NewChannelError(
			amqp.AccessRefused,
			fmt.Sprintf("exchange name '%s' contains reserved prefix 'amq.*'", method.Exchange),
			method.ClassIdentifier(),
			method.MethodIdentifier(),
		)
	}

	newExchange := exchange.New(
		method.Exchange,
		exTypeId,
		method.Durable,
		method.AutoDelete,
		method.Internal,
		false,
		method.Arguments,
	)

	if existingExchange != nil {
		if err := existingExchange.EqualWithErr(newExchange); err != nil {
			return amqp.NewChannelError(
				amqp.PreconditionFailed,
				err.Error(),
				method.ClassIdentifier(),
				method.MethodIdentifier(),
			)
		}
		channel.SendMethod(&amqp.ExchangeDeclareOk{})
		return nil
	}

	channel.conn.getVirtualHost().AppendExchange(newExchange)
	channel.SendMethod(&amqp.ExchangeDeclareOk{})

	return nil
}