# go-sse-nats-nginx

This repository contains simple setup for a real-time update system using Server-Sent Events (SSE) with NATS and NGINX.

### Run with Docker Compose

docker-compose up

### Access the Application

http://localhost?room_id=123

### Broadcast Data via API

curl -i -XPOST localhost/api/publish -d '{"room_id": "123", "message": "hi room 123"}'