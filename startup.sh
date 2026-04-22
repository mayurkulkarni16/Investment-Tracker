#!/bin/bash

echo "Starting backend..."
(cd backend && go run main.go) &

echo "Starting frontend..."
(cd frontend && npm start) &

wait