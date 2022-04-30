package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const powerRegex = "^shellies/shellyplug-s-.{6}/relay/0/power$"
const energyRegex = "^shellies/shellyplug-s-.{6}/relay/0/energy$"
const temperatureRegex = "^shellies/shellyplug-s-.{6}/temperature$"
const overtemperatureRegex = "^shellies/shellyplug-s-.{6}/overtemperature$"

var powerRegexp *regexp.Regexp
var energyRegexp *regexp.Regexp
var temperatureRegexp *regexp.Regexp
var overtemperatureRegexp *regexp.Regexp

var mqttHost string
var mqttPort string

func getShellyID(s string) (string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 0 {
		return parts[1], nil
	}
	return "", errors.New("failed to get shelly id")
}

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	topic := msg.Topic()
	id, err := getShellyID(topic)
	if err != nil {
		return
	}

	payload := string(msg.Payload())
	value, err := strconv.ParseFloat(payload, 32)
	if err != nil {
		return
	}

	if powerRegexp.MatchString(topic) {
		shellyPower.WithLabelValues(id).Set(value)
	} else if energyRegexp.MatchString(topic) {
		shellyEnergy.WithLabelValues(id).Set(value)
	} else if temperatureRegexp.MatchString(topic) {
		shellyTemperature.WithLabelValues(id).Set(value)
	} else if overtemperatureRegexp.MatchString(topic) {
		shellyOverTemperature.WithLabelValues(id).Set(value)
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	time.Sleep(time.Second * 1)
	fmt.Printf("connection lost: %v", err)
	log.Fatal(err)
}

var shellyTemperature = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "shellyplug_s_temperature",
		Help: "Reports internal device temperature in celsius",
	}, []string{
		"id",
	},
)

var shellyOverTemperature = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "shellyplug_s_overtemperature",
		Help: "Reports 1 when device has overheated, normally 0",
	}, []string{
		"id",
	},
)

var shellyPower = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "shellyplug_s_power",
		Help: "Instantaneous power consumption rate in Watts",
	}, []string{
		"id",
	},
)

var shellyEnergy = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "shellyplug_s_energy",
		Help: "Amount of energy consumed in Watt-minutes",
	}, []string{
		"id",
	},
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	parseRegexp()

	if !env("MQTT_HOST", &mqttHost) {
		mqttHost = "127.0.0.1"
	}

	if !env("MQTT_PORT", &mqttPort) {
		mqttPort = "1883"
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%s", mqttHost, mqttPort))
	opts.SetClientID("go_mqtt_client2")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	client := mqtt.NewClient(opts)

	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	_ = client.Subscribe("shellies/+/#", byte(0), messagePubHandler)

	prometheus.MustRegister(shellyEnergy)
	prometheus.MustRegister(shellyPower)
	prometheus.MustRegister(shellyTemperature)
	prometheus.MustRegister(shellyOverTemperature)

	http.Handle("/metrics", promhttp.HandlerFor(
		prometheus.DefaultGatherer,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	))
	log.Fatal(http.ListenAndServe(":9874", nil))
}

func parseRegexp() {
	var err error
	powerRegexp, err = regexp.Compile(powerRegex)
	if err != nil {
		log.Fatal(err)
	}

	energyRegexp, err = regexp.Compile(energyRegex)
	if err != nil {
		log.Fatal(err)
	}

	temperatureRegexp, err = regexp.Compile(temperatureRegex)
	if err != nil {
		log.Fatal(err)
	}

	overtemperatureRegexp, err = regexp.Compile(overtemperatureRegex)
	if err != nil {
		log.Fatal(err)
	}
}

func env(e string, v *string) bool {
	s, exists := os.LookupEnv(e)
	if !exists {
		return false
	}

	*v = s

	return true
}
