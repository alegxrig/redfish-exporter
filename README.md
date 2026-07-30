# redfish-exporter

Prometheus-экспортёр метрик серверного железа (питание, health, температура) через Redfish API (DMTF).

## Быстрый старт
```bash
docker compose up
```
Prometheus: http://localhost:9090 (target `redfish-exporter` должен быть `UP`)
Сырые метрики: http://localhost:9812/metrics

## Без docker-compose (режим разработки)
```bash
docker run --rm -p 8000:8000 dmtf/redfish-mockup-server:latest
REDFISH_ENDPOINT=http://127.0.0.1:8000 REDFISH_USER=root REDFISH_PASSWORD=password go run .
curl localhost:9812/metrics
```

## Метрики
| Метрика | Лейблы | Значение |
|---|---|---|
| `redfish_exporter_up` | — | 1, пока процесс жив |
| `redfish_system_power_on` | `system` | 1, если PowerState == On |
| `redfish_system_health_ok` | `system` | 1, если Health == OK |
| `redfish_temperature_celsius` | `chassis`, `sensor` | показание датчика, °C |

## Конфигурация
| Переменная | По умолчанию | Назначение |
|---|---|---|
| `REDFISH_ENDPOINT` | `http://127.0.0.1:8000` | адрес Redfish-сервиса (BMC или mock) |
| `REDFISH_USER` | `root` | логин |
| `REDFISH_PASSWORD` | `password` | пароль |

## Архитектура (5 строк)
1. Горутина с `time.Ticker` каждые 15с опрашивает Redfish (BMC или mock) через `gofish`.
2. Результат складывается в `snapshot` — кэш в памяти, защищённый `sync.RWMutex`.
3. Кастомный `prometheus.Collector` на каждый scrape читает кэш под `RLock()`, не трогая сеть.
4. `/metrics` отдаёт метрики выше с лейблами по системе/шасси/датчику.
5. Prometheus scrape'ит `/metrics` по расписанию из `prometheus.yml`; при недоступном BMC отдаются последние известные значения.

# redfish-exporter

Prometheus-экспортёр метрик серверного железа (питание, health, температура) через Redfish API (DMTF). Опрашивает BMC по таймеру, отдаёт закэшированный снимок на /metrics — сеть никогда не блокирует сам scrape.

## Архитектура (5 строк)
1. Горутина с `time.Ticker` каждые 15с опрашивает Redfish (BMC или mock) через `gofish`.
2. Результат складывается в `snapshot` — кэш в памяти, защищённый `sync.RWMutex`.
3. Кастомный `prometheus.Collector` на каждый scrape читает кэш под `RLock()`, не трогая сеть.
4. `/metrics` отдаёт метрики с лейблами по системе/шасси/датчику.
5. В Kubernetes Prometheus и mock BMC — отдельные поды, экспортёр находит их через кластерный DNS (Service).

## Быстрый старт

### Вариант 1 — Kubernetes (kind + Helm), основной способ
```bash
kind create cluster --name devops-lab
docker build -t redfish-exporter:local .
kind load docker-image redfish-exporter:local --name devops-lab
cd redfish-exporter-chart
helm install redfish .
kubectl port-forward svc/redfish-exporter 9812:9812
```
Метрики: http://localhost:9812/metrics

Изменить параметры без правки YAML:
```bash
helm upgrade redfish . --set exporter.replicas=2
```

### Вариант 2 — docker compose (без Kubernetes)
```bash
docker compose up --build
```
Prometheus: http://localhost:9090/targets

### Вариант 3 — локально, без контейнеров (разработка)
```bash
docker run --rm -p 8000:8000 dmtf/redfish-mockup-server:latest
REDFISH_ENDPOINT=http://127.0.0.1:8000 REDFISH_USER=root REDFISH_PASSWORD=password go run .
curl localhost:9812/metrics
```

## Метрики
| Метрика | Лейблы | Значение |
|---|---|---|
| `redfish_exporter_up` | — | 1, пока процесс жив |
| `redfish_system_power_on` | `system` | 1, если PowerState == On |
| `redfish_system_health_ok` | `system` | 1, если Health == OK |
| `redfish_temperature_celsius` | `chassis`, `sensor` | показание датчика, °C |

## Конфигурация
| Переменная (docker/go run) | Helm-параметр | По умолчанию | Назначение |
|---|---|---|---|
| `REDFISH_ENDPOINT` | `redfish.endpoint` | `http://127.0.0.1:8000` | адрес Redfish-сервиса (BMC или mock) |
| `REDFISH_USER` | `redfish.user` | `root` | логин |
| `REDFISH_PASSWORD` | `redfish.password` | `password` | пароль |
| — | `exporter.replicas` | `1` | число реплик в Kubernetes |
| — | `exporter.resources` | см. `values.yaml` | CPU/memory requests и limits |
| — | `mock.enabled` | `true` | разворачивать ли mock BMC вместе с экспортёром |

## Структура проекта
.
├── main.go # экспортёр: Collector, poller, HTTP-сервер
├── main_test.go # unit-тесты Collect()
├── cmd/probe/ # одноразовый CLI для ручной проверки Redfish-клиента
├── Dockerfile # multi-stage сборка
├── docker-compose.yml # локальный демо-стенд без k8s
├── prometheus.yml # конфиг для docker-compose варианта
└── redfish-exporter-chart/ # Helm chart для Kubernetes
├── Chart.yaml
├── values.yaml
└── templates/
├── exporter.yaml # Deployment + Service экспортёра
├── mock.yaml # Deployment + Service mock BMC (опционально)
└── config.yaml # ConfigMap + Secret


## Тесты
```bash
go test ./... -v
```
