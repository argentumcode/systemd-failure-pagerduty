package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

const errorTouchPath = "/var/tmp/failure-systemd-failure-notification"

func main() {
	if httpProxy := os.Getenv("HTTP_PROXY"); httpProxy != "" {
		proxyUrl, err := url.Parse(httpProxy)
		if err == nil {
			http.DefaultClient = &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyUrl),
				},
			}
		}
	}
	hostName, err := os.Hostname()
	if err != nil {
		log.Printf("failed to get host name: %+v", err)
		hostName = "unknown"
	}
	unitName := os.Args[1]
	cmd := exec.Command("systemctl", "status", "--full", unitName)
	out := bytes.NewBuffer(nil)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("failed to get systemd status: %+v", err)
	}
	event := pagerduty.V2Event{
		Action:     "trigger",
		Client:     "systemd-failure-notification",
		RoutingKey: os.Getenv("PAGERDUTY_ROUTING_KEY"),
	}
	details := make(map[string]string)
	details["status"] = out.String()
	payload := &pagerduty.V2Payload{
		Summary:   fmt.Sprintf("[%s] systemd failure %s", hostName, unitName),
		Timestamp: time.Now().Format(time.RFC3339),
		Severity:  "critical",
		Source:    hostName,
		Component: unitName,
		Details:   details,
	}
	event.Payload = payload
	if _, err := pagerduty.ManageEventWithContext(context.Background(), event); err != nil {
		log.Printf("failed to send to pagerduty: %+v", err)
		f, err := os.Create(errorTouchPath)
		if err != nil {
			log.Printf("failed to touch error file: %+v", err)
			return
		}
		if err := f.Close(); err != nil {
			log.Printf("failed to touch error file: %+v", err)
		}
	}
}
