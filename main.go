package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"text/template"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

const RETRY_COUNT = 6
const errorTouchPath = "/var/tmp/failure-systemd-failure-notification"

const DEFAULT_SUMMARY_TEMPLATE = "[{{.HostName}}] systemd failure {{.UnitName}}"

func generateSummary(templateText string, hostName string, unitName string) string {
	tmpl := template.New("summary")
	tmpl, err := tmpl.Parse(templateText)
	if err != nil {
		log.Printf("failed to parse summary template. use default summary: %+v", err)
		return fmt.Sprintf("[%s] systemd failure %s", hostName, unitName)
	}
	output := bytes.Buffer{}
	err = tmpl.Execute(&output, struct {
		HostName string
		UnitName string
	}{
		HostName: hostName,
		UnitName: unitName,
	})
	if err != nil {
		log.Printf("failed to execute summary template. use default summary: %+v", err)
		return fmt.Sprintf("[%s] systemd failure %s", hostName, unitName)
	}
	return output.String()
}

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
	summaryTemplate := os.Getenv("SUMMARY_TEMPLATE")
	if summaryTemplate == "" {
		summaryTemplate = DEFAULT_SUMMARY_TEMPLATE
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
		Summary:   generateSummary(summaryTemplate, hostName, unitName),
		Timestamp: time.Now().Format(time.RFC3339),
		Severity:  "critical",
		Source:    hostName,
		Component: unitName,
		Details:   details,
	}
	event.Payload = payload
	event.DedupKey = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%s-%s", hostName, unitName))))

	var retryErr error
	retryWaitDuration := time.Second
	for retryCount := 0; retryCount <= RETRY_COUNT; retryCount += 1 {
		if retryCount > 0 {
			time.Sleep(retryWaitDuration)
			retryWaitDuration *= 2
		}

		if _, err := pagerduty.ManageEventWithContext(context.Background(), event); err != nil {
			log.Printf("failed to send to pagerduty: %+v", err)
			retryErr = err
		} else {
			retryErr = nil
			break
		}
	}

	if retryErr != nil {
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
