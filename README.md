# thermia-tibber
Make hotwater the cheapest hour. 


### example systemd service file

```
[Unit]
Description=thermia tibber
[Service]
Type=simple
Restart=always
RestartSec=10s
ExecStart=/usr/local/bin/thermia-tibber
EnvironmentFile=/etc/thermia-tibber.conf
WorkingDirectory=/tmp
[Install]
WantedBy=multi-user.target

```
