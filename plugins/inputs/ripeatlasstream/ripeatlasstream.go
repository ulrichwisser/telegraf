package ripeatlasstream

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/parsers"

	gosocketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
)

type RipeAtlasStream struct {
	Servers  []string
	Topics   []string
	Username string
	Password string

	parser parsers.Parser

	// Legacy metric buffer support
	MetricBuffer int

	PersistentSession bool
	ClientID          string `toml:"client_id"`

	// Path to CA file
	SSLCA string `toml:"ssl_ca"`
	// Path to host cert file
	SSLCert string `toml:"ssl_cert"`
	// Path to cert key file
	SSLKey string `toml:"ssl_key"`
	// Use SSL but skip chain & host verification
	InsecureSkipVerify bool

	sync.Mutex
	client *gosocketio.Client
	done   chan struct{}

	// keep the accumulator internally:
	acc telegraf.Accumulator

	started bool
}

var sampleConfig = `
  servers = ["localhost:1883"]
  ## MQTT QoS, must be 0, 1, or 2
  qos = 0

  ## Topics to subscribe to
  topics = [
    "telegraf/host01/cpu",
    "telegraf/+/mem",
    "sensors/#",
  ]

  # if true, messages that can't be delivered while the subscriber is offline
  # will be delivered when it comes back (such as on service restart).
  # NOTE: if true, client_id MUST be set
  persistent_session = false
  # If empty, a random client ID will be generated.
  client_id = ""

  ## username and password to connect MQTT server.
  # username = "telegraf"
  # password = "metricsmetricsmetricsmetrics"

  ## Optional SSL Config
  # ssl_ca = "/etc/telegraf/ca.pem"
  # ssl_cert = "/etc/telegraf/cert.pem"
  # ssl_key = "/etc/telegraf/key.pem"
  ## Use SSL but skip chain & host verification
  # insecure_skip_verify = false

  ## Data format to consume.
  ## Each data format has it's own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_INPUT.md
  data_format = "influx"
`

func (m *RipeAtlasStream) SampleConfig() string {
	return sampleConfig
}

func (m *RipeAtlasStream) Description() string {
	return "Read events from a Ripe Atlas Stream"
}

type Subscription struct {
	Streamtype string `json:"stream_type"`
	//Prb        int    `json:"prb"`
}

type Probestatus struct {
	Timestamp  int    `json:"timestamp"`
	Prefix     string `json:"prefix"`
	Event      string `json:"event"`
	Controller string `json:"controller"`
	Id         int    `json:"prb_id"`
	Type       string `json:"type"`
	Asn        ASN    `json:"asn"`
}

type ASN struct {
	ID string
}

func (a *ASN) UnmarshalJSON(b []byte) (err error) {
	var v interface{}

	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch v.(type) {
	case string:
		a.ID = v.(string)
	case int:
		a.ID = fmt.Sprintf("%d", v.(int))
	case float32:
		a.ID = fmt.Sprintf("%.0f", v.(float32))
	case float64:
		a.ID = fmt.Sprintf("%.0f", v.(float64))
	case nil:
		a.ID = ""
	default:
		return fmt.Errorf("Unknown data type for ASN")
	}
	return nil
}

func (m *RipeAtlasStream) Start(acc telegraf.Accumulator) error {
	m.Lock()
	defer m.Unlock()
	m.started = false
	m.acc = acc
	var err error

	//connect to server, you can use your own transport settings
	m.client, err = gosocketio.Dial(
		gosocketio.GetUrl("atlas-stream.ripe.net", 80, false, "/stream"),
		transport.GetDefaultWebsocketTransport(),
	)
	if err != nil {
		return err
	}

	m.client.On(gosocketio.OnConnection, func(h *gosocketio.Channel) {
		fmt.Println("Connected")
	})
	m.client.Emit("atlas_subscribe", Subscription{Streamtype: "probestatus"})
	m.client.On("atlas_probestatus", func(h *gosocketio.Channel, args Probestatus) {
		fmt.Printf("New status for probe %d: %s\n", args.Id, args.Event)
		tags := map[string]string{
			"prefix":     args.Prefix,
			"controller": args.Controller,
			"probe_id":   fmt.Sprintf("%d", args.Id),
			"type":       args.Type,
			"asn":        args.Asn.ID,
		}
		if tags["asn"] == "" {
			tags["asn"] = "0"
		}
		if tags["prefix"] == "" {
			tags["prefix"] = "0"
		}
		fields := map[string]interface{}{"status": args.Event}
		acc.AddFields("atlas_probestatus", fields, tags, time.Unix(int64(args.Timestamp), 0))
	})
	//m.client.Emit("atlas_subscribe", "")

	m.done = make(chan struct{})

	return nil
}

func (m *RipeAtlasStream) Gather(acc telegraf.Accumulator) error {
	return nil
}

func (m *RipeAtlasStream) Stop() {
	m.Lock()
	defer m.Unlock()
}

func init() {
	inputs.Add("ripeatlasstream", func() telegraf.Input { return &RipeAtlasStream{} })
}
