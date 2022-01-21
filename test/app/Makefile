repo = k3d-local-registry:50442
namespace = test

build:
	docker build -t hello:latest .

deploy:
	helm upgrade --install example chart --set image.repository=hello --set image.tag=latest