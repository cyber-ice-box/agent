proto:
	docker-compose -f docker-compose.dev.yml run --rm protobufCompiler

push:
	docker image build -t cybericebox/agent .
	docker push cybericebox/agent
