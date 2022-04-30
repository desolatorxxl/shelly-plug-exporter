# shelly plug s MQTT exporter

## Usage

Start mosquitto MQTT broker:

```sh
podman run -it -p 1883:1883 -p 9001:9001 -v $PWD/mosquitto.conf:/mosquitto/config/mosquitto.conf eclipse-mosquitto
```

Run the exporter:

```sh
podman build -t shell-plug-exporter .
podman run -e MQTT_HOST=192.167.178.123 -p 9874:9874 shelly-plug-exporter
```
Configure prometheus:

```yaml
scrape_configs:
- job_name: shelly
  static_configs:
  - targets: ['192.168.178.123:9874']
```

Watch mqtt:

```sh
mosquitto_sub -v -t 'shellies/+/#'
```
