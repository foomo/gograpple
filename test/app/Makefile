repo=local-registry:56523

build:
	docker build -t $(repo)/hello:latest .
	-docker push $(repo)/hello:latest

deploy:
	helm upgrade --install example chart --set image.repository=$(repo)/hello --set image.tag=latest
