# Coping

Dumb co-operative uptime monitoring.

## Usage

```
Usage of ./coping:
  -alertCount int
    	How many times a service can report failure before alerting (default 3)
  -buddies string
    	Comma-separated list of buddies to use for bootstrapping
  -checkInterval int
    	How often to check services (in seconds) (default 60)
  -port int
    	Port to listen on (default 9999)
  -services string
    	Comma-separated list of services to check
  -syncInterval int
    	How often to sync services/buddies state (in seconds) (default 60)
```

## Examples

Single node (warn immediately if the state changes via `notify-send`):

```
./coping -alertCount 1 -checkInterval 60 -services "http://example.org" | xargs -l1 notify-send Coping
```

Multiple nodes (warn after 3 checks and share services):

```
./coping -port 9999 -alertCount 3 -checkInterval 30 -services "http://example.org,http://example.com" > alerts.txt
```

```
./coping -port 8888 -alertCount 3 -checkInterval 30 -services "http://example.net" -buddies "http://127.0.0.1:9999" > alerts.txt
```
