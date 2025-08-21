# Concept insight MCP

This project is based on [Foxy context - streamable http](https://github.com/strowk/foxy-contexts/tree/6783020204467a1834d31fc35c9ed247a531bfe8/examples/streamable_http)

## Prerequisite 
- go 
- n8n
- create the .env with the correct slack token (ask @Alexis, see .env.example)

## Start
- to start the project:

```bash
go run main.go
```

## n8n
- you can run n8n in docker using (don't forget to create the certificates):
```bash
docker run -it --rm \
--name n8n \
-p 5678:5678 \
-e GENERIC_TIMEZONE="Europe/Berlin" \
-e TZ="Europe/Berlin" \
-e N8N_ENFORCE_SETTINGS_FILE_PERMISSIONS=true \
-e N8N_RUNNERS_ENABLED=true \
-e N8N_PROTOCOL=https \
-e N8N_SSL_KEY=/home/node/.n8n/cert/privkey.pem \
-e N8N_SSL_CERT=/home/node/.n8n/cert/cert.pem \
-v ~/n8n_certs:/home/node/.n8n/cert \
-v n8n_data:/home/node/.n8n \
docker.n8n.io/n8nio/n8n
```
- To create certificates, in the fodler where you are running the previous docker command: 
```bash
openssl req -x509 -newkey rsa:4096 -keyout privkey.pem -out cert.pem -sha256 -days 365 -nodes -subj "/CN=localhost"
```
