#!/bin/bash

scp http-proxy lantern@$1:http-proxy-new
ssh lantern@$1 "sudo service http-proxy stop && cp /home/lantern/http-proxy-new /home/lantern/http-proxy && sudo setcap 'cap_net_bind_service=+ep' /home/lantern/http-proxy && sudo service http-proxy start"
