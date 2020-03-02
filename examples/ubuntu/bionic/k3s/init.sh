#!/bin/bash
env | grep 'K3S' > /etc/systemd/system/k3s.service.env
exec "$@"
