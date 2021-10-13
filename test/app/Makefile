repo = k3d-local-registry:50442
namespace = test

build:
	docker build -t $(repo)/hello:latest .
	docker push $(repo)/hello:latest

deploy:
	helm upgrade --install example chart --set image.repository=$(repo)/hello --set image.tag=latest -n $(namespace)