package inventory

import (
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
	"time"
)

var (
	queueLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "smfg_inventory_queue_latency",
			Help:       "The latency quantiles for the given queue request",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"exchange", "queue"},
	)

	queueVolume = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smfg_inventory_queue_volume",
			Help: "Number of times a record was written to the queue",
		},
		[]string{"exchange", "queue"},
	)

	queueErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "smfg_inventory_queue_errors",
			Help: "Number of times an error occurred writing to a queue",
		},
		[]string{"exchange", "queue"},
	)
)

type Metric struct {
	exchange string
	queue string
	start time.Time
}

func StartMetric(exchange, queue string) *Metric {
	queueVolume.WithLabelValues(exchange, queue).Inc()
	return &Metric{exchange: exchange, queue: queue, start: time.Now()}
}

func (m *Metric) Complete(err error) {
	if err != nil {
		queueErrors.WithLabelValues(m.exchange, m.queue).Inc()
	}
	dur := float64(time.Since(m.start).Milliseconds())
	queueLatency.WithLabelValues(m.exchange, m.queue).Observe(dur)
}

func init() {
	prometheus.MustRegister(queueVolume)
	prometheus.MustRegister(queueLatency)
	prometheus.MustRegister(queueErrors)
}


type Queue interface {
	Send(body interface{}, options ...MessageOption) error
	Close() (error, error)
}

type rabbitClient struct {
	conn *amqp.Connection
	ch *amqp.Channel
	q *amqp.Queue
}

type MessageOption func(m *messageOptions)

type messageOptions struct {
	exchange string
	routingKey string
	mandatory bool
	immediate bool
}

func (r *rabbitClient) Close() (chErr error, cnErr error) {
	chErr = r.ch.Close()
	cnErr = r.conn.Close()
	return chErr, cnErr
}

func (r *rabbitClient) Send(body interface{}, options ...MessageOption) error {
	mo := &messageOptions{}
	for _, option := range options {
		option(mo)
	}

	j, err := json.Marshal(body)
	if err != nil {
		return err
	}

	m := StartMetric(mo.exchange, r.q.Name)
	err = r.ch.Publish(
		mo.exchange,
		r.q.Name,
		mo.mandatory,
		mo.immediate,
		amqp.Publishing{
			ContentType: "application/json",
			Body: j,
		})
	if err != nil {
		m.Complete(err)
		return err
	}

	m.Complete(nil)
	return nil
}

func Exchange(x string) func(m *messageOptions) {
	return func(m *messageOptions) {
		m.exchange = x
	}
}

func RoutingKey(rk string) func(m *messageOptions) {
	return func(m *messageOptions) {
		m.routingKey = rk
	}
}

func Mandatory(m *messageOptions) {
	m.mandatory = true
}

func Immediate(m *messageOptions) {
	m.immediate = true
}

type queueOptions struct {
	durable bool
	deleteUnused bool
	exclusive bool
	noWait bool
}

type QueueOption func(q *queueOptions)

func NewRabbitClient(qName, user, pass, host, port string, options ...QueueOption) (*rabbitClient, error) {
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%s", user, pass, host, port)
	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	qo := &queueOptions{}

	for _, option := range options {
		option(qo)
	}

	q, err := ch.QueueDeclare(qName, qo.durable, qo.deleteUnused, qo.exclusive, qo.noWait, nil)
	if err != nil {
		return nil, err
	}

	return &rabbitClient{conn: conn, ch: ch, q: &q}, nil
}

func Durable(q *queueOptions) {
	q.durable = true
}

func DeleteUnused(q *queueOptions) {
	q.deleteUnused = true
}

func Exclusive(q *queueOptions) {
	q.exclusive = true
}

func NoWait(q *queueOptions) {
	q.noWait = true
}

type MockQueue struct {
	SendFunc func(body interface{}, options ...MessageOption) error
	CloseFunc func() (error, error)
}

func (r MockQueue) Send(body interface{}, options ...MessageOption) error {
	return r.SendFunc(body, options...)
}

func (r MockQueue) Close() (error, error) {
	return r.CloseFunc()
}

func NewMockQueue() MockQueue {
	return MockQueue{
		SendFunc:  func(body interface{}, options ...MessageOption) error { return nil },
		CloseFunc: func() (error, error) { return nil, nil },
	}
}