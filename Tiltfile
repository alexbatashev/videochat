# Monitoring
k8s_yaml('k8s/monitoring.yaml')

load('ext://helm_remote', 'helm_remote')
helm_remote(
  'kube-prometheus-stack', 
  repo_url='https://prometheus-community.github.io/helm-charts', 
  namespace='monitoring'
)

k8s_yaml([
  'k8s/sfu-monitoring.yaml',
  'k8s/turn-monitoring.yaml',
  'k8s/clickhouse-monitoring.yaml',
  'k8s/prometheus-role.yaml',
  'k8s/prometheus-pod.yaml'
])

# Service

k8s_yaml([
  # 'k8s/grafana-pod.yaml',
  # 'k8s/grafana-service.yaml',
  'k8s/nginx-deployment.yaml',
  'k8s/nginx-service.yaml',
  'k8s/rabbitmq-configmap.yaml',
  'k8s/rabbitmq-credentials.yaml',
  'k8s/rabbitmq-service.yaml',
  'k8s/rabbitmq-statefulset.yaml',
  'k8s/rest-deployment.yaml',
  'k8s/rest-service.yaml',
  'k8s/service-reader-role-bindings.yaml',
  'k8s/service-reader-role.yaml',
  'k8s/sfu-pod.yaml',
  'k8s/turn-deployment.yaml',
  'k8s/turn-service.yaml',
  'k8s/clickhouse-configmap.yaml',
  'k8s/zookeeper.yaml',
  'k8s/tarantool-cluster.yaml',
  'k8s/clickhouse-operator-install.yaml',
  'k8s/clickhouse-cluster.yaml'
])

k8s_yaml(helm('charts/tarantool-operator'))
k8s_kind('ReplicasetTemplate', image_json_path='{.spec.template.spec.containers[0].image}')
k8s_kind('ClickHouseInstallation', image_json_path='{.spec.templates.podTemplates[0].spec.containers[0].image}')

docker_build('itcoursevideochat/rest', 'rest_server')
docker_build('itcoursevideochat/sfu', 'sfu_server')
docker_build('itcoursevideochat/tarantool', '.', dockerfile='Dockerfile.tarantool')
docker_build('itcoursevideochat/turn', 'turn')
docker_build('itcoursevideochat/nginx', '.', dockerfile='Dockerfile.nginx')
docker_build('itcoursevideochat/clickhouse', '.', dockerfile="Dockerfile.clickhouse")
