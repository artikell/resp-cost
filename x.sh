#!/bin/bash
python3 testing.py --target valkey@9.0 --password respcost --addr localhost:7390
python3 testing.py --target valkey@8.1 --password respcost --addr localhost:7381
python3 testing.py --target redis@5.0 --password respcost --addr localhost:6350
python3 testing.py --target redis@6.0 --password respcost --addr localhost:6360
python3 testing.py --target redis@7.0 --password respcost --addr localhost:6370
python3 testing.py --target redis@8.0 --password respcost --addr localhost:6380