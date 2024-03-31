# Bunq Firefly III sync

The bunq Firefly III sync loads all bunq transactions via de bunq api and pushed them to the chosen Firefly III instance.

## Configuration

Configuration is done using environment variables:

| Environment variable | Default | Description |
| -------------------- | ------- | ----------- |
| STORAGE_LOCATION | ./storage/ | Location where the bunq FireFly III can persist files |
| BUNQ_API_BASE_URL | https://public-api.sandbox.bunq.com/v1 |
| BUNQ_API_KEY | | |
| BUNQ_PRIVATE_KEY_FILE_NAME | bunq_client.key | |
| BUNQ_PUBLIC_KEY_FILE_NAME | bunq_client.pub.key | |
| BUNQ_INSTALLATION_FILE_NAME | bunq_installation.json | |
| BUNQ_DEVICE_SERVER_FILE_NAME | bunq_device_server.json | |
| BUNQ_SESSION_SERVER_FILE_NAME | bunq_session_server.json | |
| BUNQ_USER_AGENT | BunqFireflySync/1.0 | |
| BUNQ_PERMITTED_IPS | * | Comma-separated list with all ips that are allowed to use the bunq api key |
