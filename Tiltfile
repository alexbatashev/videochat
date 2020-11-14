k8s_yaml([
  'k8s/grafana-pod.yaml',
  'k8s/grafana-service.yaml',
  'k8s/mongo-pod.yaml',
  'k8s/mongo-service.yaml',
  'k8s/nginx-deployment.yaml',
  'k8s/nginx-service.yaml',
  'k8s/rabbitmq-pod.yaml',
  'k8s/rabbitmq-service.yaml',
  'k8s/rest-deployment.yaml',
  'k8s/rest-service.yaml',
  'k8s/service-reader-role-bindings.yaml',
  'k8s/service-reader-role.yaml',
  'k8s/sfu-pod.yaml',
  'k8s/tarantool-pod.yaml',
  'k8s/tarantool-service.yaml',
  'k8s/turn-deployment.yaml',
  'k8s/turn-service.yaml'
])

docker_build('itcoursevideochat/rest', 'rest_server')
docker_build('itcoursevideochat/sfu', 'sfu_server')
docker_build('itcoursevideochat/tarantool', '.', dockerfile='Dockerfile.tarantool')
docker_build('itcoursevideochat/grafana', '.', dockerfile='Dockerfile.grafana')
docker_build('itcoursevideochat/turn', 'turn')
docker_build('itcoursevideochat/nginx', '.', dockerfile='Dockerfile.nginx')
