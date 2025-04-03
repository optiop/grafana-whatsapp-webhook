# Grafana Whatsapp Webhook
[![Go Report Card](https://goreportcard.com/badge/github.com/optiop/grafana-whatsapp-webhook)](https://goreportcard.com/report/github.com/optiop/grafana-whatsapp-webhook)
[![Slack Community](https://badgen.net/badge/icon/slack?icon=slack&label)](https://join.slack.com/t/optioporg/shared_invite/zt-33axtzuao-Kd5NzaVm2GOhozBHOTj_Yg)

This repository acts as a webhook service that listens for Grafana alerts 
and forwards them to a specified WhatsApp number or group. It enables 
seamless integration of Grafana alerts with WhatsApp for real-time notifications.

## Usage
1. Run the service using `docker` command.
  ```bash
  docker run -p 8080:8080 -e WHATSAPP_APP_TOKEN=secure_token -v ./out:/app/out/ -d \
    ghcr.io/optiop/grafana-whatsapp-webhook:latest
  ```

2. From the WhatsApp menu, select `Linked devices`, and choose `Link a device`. Scan
the QR code created at `./out/qr.png`.

3. In Grafana create a contact point with the following URL:
  ```
  http://<your_ip>:8080/whatsapp/send/grafana-alert/user/<phone_number>/<WHATSAPP_APP_TOKEN>
  http://<your_ip>:8080/whatsapp/send/grafana-alert/group/<group_id>/<WHATSAPP_APP_TOKEN>
  ```


> **⚠️ WARNING:** If you stop using this service, ensure you unlink 
> the device from WhatsApp to maintain your account's security.
