#!/bin/bash
python3 testing.py --target valkey@8.1 --password respcost --addr localhost:7379
python3 testing.py --target redis@5.0 --password respcost --addr localhost:7380
python3 testing.py --target redis@6.0 --password respcost --addr localhost:7381
python3 testing.py --target redis@7.0 --password respcost --addr localhost:7382