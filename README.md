# systemd failure pagerduty

This is a small program to forward systemd failure notifications to PagerDuty.

## Environments Variables

`PAGERDUTY_ROUTING_KEY` is a required variable and must be set to PagerDuty Event API v2 Routing Key.   
`HTTP_PROXY` is an optional variable and should be set to the HTTP Proxy URL.

## Setup

Install systemd-failure-pagerduty to /bin or your preferred directory.

Create a systemd unit `/etc/systemd/system/systemd-failure-pagerduty@.service` for this program.
```
[Unit]
Description=notify systemd failure to PagerDuty
After=network.target

[Service]
Type=simple
ExecStart=/bin/systemd-failure-pagerduty %i
EnvironmentFile=/etc/default/systemd-failure-pagerduty
User=nobody
Group=nogroup
```

Write the necessary environment variables in `/etc/default/systemd-failure-pagerduty`.
```
PAGERDUTY_ROUTING_KEY=...
HTTP_PROXY=...
```

Set `systemd-failure-pagerduty@%n` to the OnFailure hook of the service for which you want to be notified of failures.

